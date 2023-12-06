package spiproto

import (
	"errors"
)

type Conn interface {
	TxRx(tx, rx []byte) error
}

type Proto struct {
	conn Conn
	buf  []byte
}

func New(c Conn) *Proto {
	return &Proto{conn: c, buf: make([]byte, 16)}
}
func (d *Proto) Reset() error {
	return d.runCmd(0xC0, None, nil, 0)
}

func (d *Proto) Read(a Addr, buf []byte) error {
	err := d.runCmd(0x03, a, nil, len(buf))
	if err != nil {
		return err
	}
	copy(buf, d.buf[2:])
	return nil
}

type Status byte

func (st Status) Rx0Int() bool {
	return (st & (1 << 0)) != 0
}

func (st Status) Rx1Int() bool {
	return (st & (1 << 1)) != 0
}

func (d *Proto) ReadStatus() (Status, error) {
	err := d.runCmd(0xA0, None, nil, 1)
	if err != nil {
		return 0, err
	}
	return Status(d.buf[1]), nil
}

type RxStatus byte

func (st RxStatus) MsgInRxBuf0() bool {
	return (st & (1 << 6)) != 0
}

func (st RxStatus) MsgInRxBuf1() bool {
	return (st & (1 << 7)) != 0
}

func (st RxStatus) IsRemoteFrame() bool {
	return (st & (1 << 3)) != 0
}

func (st RxStatus) IsExtFrame() bool {
	return (st & (1 << 4)) != 0
}

func (d *Proto) ReadRxStatus() (RxStatus, error) {
	err := d.runCmd(0xB0, None, nil, 1)
	if err != nil {
		return 0, err
	}
	return RxStatus(d.buf[1]), nil
}

func (d *Proto) ReadRxBuf(ibuf int, buf []byte) error {
	instr := uint8(0x90)
	if ibuf == 1 {
		instr |= 1 << 2
	}
	err := d.runCmd(instr, None, nil, len(buf))
	if err != nil {
		return err
	}
	copy(buf, d.buf[1:])
	return nil
}

func (d *Proto) ReadRxData(ibuf int, buf []byte) error {
	instr := uint8(0x92)
	if ibuf == 1 {
		instr |= 1 << 2
	}
	err := d.runCmd(instr, None, nil, len(buf))
	if err != nil {
		return err
	}
	copy(buf, d.buf[1:])
	return nil
}

func (d *Proto) Write(a Addr, buf []byte) error {
	return d.runCmd(0x02, a, buf, 0)
}

func (d *Proto) LoadTxBuf(ibuf int, buf []byte) error {
	instr := uint8(0x40) | (uint8(ibuf) << 1)
	return d.runCmd(instr, None, buf, 0)
}

func (d *Proto) LoadTxData(ibuf int, buf []byte) error {
	instr := uint8(0x41) | (uint8(ibuf) << 1)
	return d.runCmd(instr, None, buf, 0)
}

type Addr uint8

const (
	None    Addr = 0xFF
	CNF3    Addr = 0x28
	CNF2    Addr = 0x29
	CNF1    Addr = 0x2A
	CANINTE Addr = 0x2B
	CANINTF Addr = 0x2C
	EFLG    Addr = 0x2D
	CANCTRL Addr = 0x2F

	RXM0SIDH Addr = 0x20
	RXM0SIDL Addr = 0x21
	RXM0EID8 Addr = 0x22
	RXM0EID0 Addr = 0x23

	TXB0CTRL Addr = 0x30
	TXB0SIDH Addr = 0x31

	RXB0CTRL Addr = 0x60
	RXB1CTRL Addr = 0x70

	// RXBnCTRL
	RXMMask = 3 << 5
	RXMAny  = 3 << 5
	BUKT    = 1 << 2

	// RXM0SIDL
	SIDMask = 7 << 5
	EIDMask = 3 << 0

	// CANCTRL
	REQOPMask   = 7 << 5
	REQOPNormal = 0 << 5

	// TXBnCTRL
	TXREQ = 1 << 3

	// TXBnSIDL
	EXIDE       = 1 << 3
	SIDLStdMask = 7<<5 | EXIDE

	// CANINTE
	RX1IE = 1 << 1
	RX0IE = 1 << 0
)

func (d *Proto) BitModify(a Addr, mask, data byte) error {
	return d.runCmd(0x05, a, []byte{mask, data}, 0)
}

func (d *Proto) runCmd(instr uint8, a Addr, tx []byte, nrx int) error {
	b := d.buf
	b[0] = instr
	n := 1
	if a != None {
		b[1] = uint8(a)
		n++
	}
	ntx := len(tx)
	if n+ntx > len(b) {
		return errors.New("tx data does not fit into msg buffer")
	}
	if ntx != 0 {
		copy(b[n:], tx)
	}
	n += ntx
	nzero := nrx - ntx
	if nzero > 0 {
		if n+nzero > len(b) {
			return errors.New("rx data does not fit into msg buffer")
		}
		for i := 0; i < nzero; i++ {
			b[n+i] = 0
		}
		n += nzero
	}
	b = b[:n]
	brx := b
	if nrx == 0 {
		brx = nil
	}
	return d.conn.TxRx(b, brx)
}
