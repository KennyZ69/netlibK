package netlibk

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
)

// Broadcast is a hardware address of a frame that should be sent to every device on given subnet
var EthernetBroadcast = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

const minPayloadLen = 46

// build the headers, now with no error check but I may do it later idk
func BuildEthernetHeader(sourceMAC, destMAC net.HardwareAddr, etherType EtherType) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(sourceMAC)
	buf.Write(destMAC)
	binary.Write(buf, binary.BigEndian, etherType)
	return buf.Bytes(), nil
}

// get the number of bytes required for this frame / eth header
func (et *EthernetHeader) length() int {
	pLen := len(et.Payload)
	// make the payload at least the mininal payload length so zero-pad it up till the min required
	if pLen < minPayloadLen {
		pLen = minPayloadLen
	}

	// 6 bytes for destination mac addr
	// 6 bytes for source mac addr
	// 2 bytes for ether type
	// N bytes for payload length (possibly padded)
	return 6 + 6 + 2 + pLen
}

// allocate a byte slice into the eth header / frame and make it to binary implementing the read func
func (et *EthernetHeader) Marshal() ([]byte, error) {
	b := make([]byte, et.length())
	_, err := et.read(b)
	return b, err
}

// unmarshal byte slice into the ethernet header / frame
func (et *EthernetHeader) Unmarshal(b []byte) error {
	// 6 + 6 + 2 is the minimal size of the byte slice and then the payload length
	if len(b) < 14 {
		// return fmt.Errorf("Error byte slice of smaller size than 14")
		return io.ErrUnexpectedEOF
	}

	n := 14
	ethType := EtherType(binary.BigEndian.Uint16(b[n-2 : n]))
	et.EtherType = ethType

	// now make a byte slice for the mac dest and source and the payload (mac + mac + payload = length)
	bb := make([]byte, 6+6+len(b[n:]))
	copy(bb[0:6], b[0:6])
	et.DestAddr = bb[0:6]
	copy(bb[6:12], b[6:12])
	et.SourceAddr = bb[6:12]

	copy(bb[12:], b[12:])
	et.Payload = bb[12:]

	return nil
}

// make the binary form of a frame or eth header for the marshal to then allocate
func (et *EthernetHeader) read(b []byte) (int, error) {
	copy(b[0:6], et.DestAddr)
	copy(b[6:12], et.SourceAddr)
	n := 12

	// for now I do not care about VLAN I guess I do not need it for goapt

	binary.BigEndian.PutUint16(b[n:n+2], uint16(et.EtherType))
	copy(b[n+2:], et.Payload)
	return len(b), nil
}
