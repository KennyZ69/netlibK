package main

import (
	"bytes"
	"encoding/binary"
	"net"
)

// build the headers, now with no error check but I may do it later idk
func BuildEthernetHeader(sourceMAC, destMAC net.HardwareAddr, etherType EtherType) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(sourceMAC)
	buf.Write(destMAC)
	binary.Write(buf, binary.BigEndian, etherType)
	return buf.Bytes(), nil
}
