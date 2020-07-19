This package provides a driver written in Go for the MCP2515 CAN controller.

Although still lacking configuration options (a bitrate of 500kBit/s is hard-coded),
and interrupt support, the driver -- as part of a development tool,
a CAN-SPI-gateway written in TinyGo and running on the BBC microbit --
was sufficient for developing an SPI bootloader.
