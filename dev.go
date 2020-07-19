package mcp2515

import (
	"errors"
	"time"

	"github.com/knieriem/can"
	"github.com/knieriem/mcp2515/spiproto"
)

type Dev struct {
	p        *spiproto.Proto
	availRx1 bool
}

func NewDevice(c spiproto.Conn) *Dev {
	p := spiproto.New(c)
	d := new(Dev)
	d.p = p
	return d
}

func (d *Dev) Init() error {
	err := d.p.Reset()
	if err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)

	// Setup bit timing.
	err = d.p.Write(spiproto.CNF3, []byte{0x01, 0xB5, 00}) // 500k
	//	err = d.p.Write(spiproto.CNF3, []byte{0x01, 0x91, 0x40}) 	// 1000k
	if err != nil {
		return err
	}

	// Enable filters, and rollover: If RXB0 is full,
	// next arriving message will be written to RXB1.
	err = d.p.BitModify(spiproto.RXB0CTRL, spiproto.RXMMask|spiproto.BUKT, spiproto.BUKT)
	if err != nil {
		return err
	}

	// Set mask 0 to zero (allow any message)
	err = d.p.Write(spiproto.RXM0SIDH, []byte{0})
	if err != nil {
		return err
	}
	err = d.p.BitModify(spiproto.RXM0SIDL, spiproto.SIDMask|spiproto.EIDMask, 0)
	if err != nil {
		return err
	}
	err = d.p.Write(spiproto.RXM0EID8, []byte{0, 0})
	if err != nil {
		return err
	}

	// Enable receive interrupts for both buffers; this most likely
	// stems from an experiment, and should be removed,
	// or turned into an option.
	err = d.p.BitModify(spiproto.CANINTE, spiproto.RX1IE|spiproto.RX0IE, spiproto.RX1IE|spiproto.RX0IE)
	if err != nil {
		return err
	}

	// Set normal operation mode.
	err = d.p.BitModify(spiproto.CANCTRL, spiproto.REQOPMask, spiproto.REQOPNormal)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dev) Read(m *can.Msg) error {
	if d.availRx1 {
		d.availRx1 = false
		return d.readRx(1, m)
	}
	st, err := d.p.ReadStatus()
	if err != nil {
		return ErrNoMsg
	}
	if st.Rx0Int() {
		if st.Rx1Int() {
			d.availRx1 = true
		}
		return d.readRx(0, m)
	}
	if st.Rx1Int() {
		return d.readRx(1, m)
	}
	return ErrNoMsg
}

func (d *Dev) readRx(i int, m *can.Msg) error {
	var buf [13]byte
	err := d.p.ReadRxBuf(i, buf[:])
	if err != nil {
		return err
	}
	id, extFrame := decodeID(buf[:])
	m.Flags = 0
	if extFrame {
		m.Flags = can.ExtFrame
	}
	m.Id = id
	m.Len = int(buf[4])
	copy(m.Data[:], buf[5:5+m.Len])
	return nil
}

func (d *Dev) setID(base spiproto.Addr, id uint32, extFrame bool) error {
	var sidh, sidl, sidlm byte

	if extFrame {
		sidlm = spiproto.SIDLStdMask | 3
		sidl = byte((id>>13)&(7<<5)) | spiproto.EXIDE | byte((id>>16)&3)
		sidh = byte(id >> 21)
		err := d.p.Write(base+2, []byte{byte(id >> 8), byte(id)})
		if err != nil {
			return err
		}
	} else {
		sidlm = spiproto.SIDLStdMask
		sidl = byte(id << 5)
		sidh = byte(id >> 3)
	}
	err := d.p.Write(base, []byte{sidh})
	if err != nil {
		return err
	}
	return d.p.BitModify(base+1, sidlm, sidl)
}

func (d *Dev) encodeID(dest []byte, id uint32, extFrame bool) {
	if extFrame {
		dest[0] = byte(id >> 21)
		dest[1] = byte((id>>13)&(7<<5)) | spiproto.EXIDE | byte((id>>16)&3)
		dest[2] = byte(id >> 8)
		dest[3] = byte(id)
		return
	}
	dest[0] = byte(id >> 3)
	dest[1] = byte(id << 5)
	dest[2] = 0
	dest[3] = 0
}

func decodeID(buf []byte) (id uint32, extFrame bool) {
	if buf[1]&spiproto.EXIDE != 0 {
		id = (uint32(buf[0]) << 21) | (uint32(buf[1]&(7<<5)) << 13) | (uint32(buf[1]&3) << 16) | (uint32(buf[2]) << 8) | uint32(buf[3])
		return id, true
	}
	id = (uint32(buf[0]) << 3) | (uint32(buf[1]) >> 5)
	return id, false
}

func (d *Dev) Write(m *can.Msg) error {
	var b [13]byte
	err := d.p.Read(spiproto.TXB0CTRL, b[:])
	if err != nil {
		return err
	}
	if (b[0] & spiproto.TXREQ) != 0 {
		return ErrTxBufNotEmpty
	}
	d.encodeID(b[:], m.Id, m.ExtFrame())
	b[4] = byte(m.Len)
	copy(b[5:], m.Data[:])
	err = d.p.LoadTxBuf(0, b[:])
	if err != nil {
		return err
	}
	return d.p.BitModify(spiproto.TXB0CTRL, spiproto.TXREQ, spiproto.TXREQ)
}

var ErrNoMsg = errors.New("no message available")
var ErrTxBufNotEmpty = errors.New("tx buffer not empty")
