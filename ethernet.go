package main

import (
	"bytes"
	"encoding/binary"
	"net"
)

// Broadcast is a hardware address of a frame that should be sent to every device on given subnet
var EthernetBroadcast = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

// build the headers, now with no error check but I may do it later idk
func BuildEthernetHeader(sourceMAC, destMAC net.HardwareAddr, etherType EtherType) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(sourceMAC)
	buf.Write(destMAC)
	binary.Write(buf, binary.BigEndian, etherType)
	return buf.Bytes(), nil
}
