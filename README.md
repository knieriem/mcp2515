This package provides a driver written in Go for the MCP2515 CAN controller.

Although still lacking configuration options (a bitrate of 500kBit/s is hard-coded),
and interrupt support, the driver -- as part of a development tool,
a CAN-SPI-gateway written in TinyGo and running on the BBC microbit --
was sufficient for developing an SPI bootloader.

### Creating a driver instance

```go
import "github.com/knieriem/mcp2515"

...

// Setup an SPI connection to the MCP2515 device,
// which may be anything satisfying the interface:
//	type Conn interface {
//		TxRx(tx, rx []byte) error
//	}
//
spiConn := ...
d := mcp2515.NewDevice(spiConn)

// Init performs a Reset, sets the Bitrate to 500kBit/s,
// configures filters to allow any message,
// and enables rollover mode.
// It would be preferable to have functional configuration options,
// but they have not been implemented yet.
err := d.Init()
if err != nil {
	return err
}

...
```

### Sending a message

```go
import "github.com/knieriem/can"
...

var msg can.Msg

msg.Id = 0x1234567
msg.Flags |= can.ExtFrame
msg.Len = 2
msg.Data[0] = ...
msg.Data[1] = ...

err = d.Write(&msg)
if err != nil {
	if err == mcp2515.ErrTxBufNotEmpty {
		// ...
	}
	return err
}

```

### Receiving a message

```go
var m can.Msg

err = d.Read(&m)
if err != nil {
	if err == mcp2515.ErrNoMsg {
		// time.Sleep(300 * time.Microsecond)
		// try again
	}
	return err
}

println("->", m.Id, m.Len, m.Data[0])
```
