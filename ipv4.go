package netlibk

import (
	"bytes"
	"encoding/binary"
	"math/rand/v2"
	"net"
)

func BuildIPv4Header(sourceIp, destIp net.IP, protocol uint16, payload []byte) ([]byte, error) {
	header := IPv4Header{
		Version:        0x45, // Version 4 and IHL 5 (20 bytes)
		Service:        0,    // Default
		TotalLen:       htons(uint16(20 + len(payload))),
		Id:             htons(uint16(rand.IntN(65535))), // random int ID
		FragmentOffset: 0,
		TTL:            64, // Default
		Protocol:       uint8(protocol),
		// calculate checksum later from buffer
	}

	copy(header.SourceIp[:], sourceIp.To4())
	copy(header.DestIp[:], destIp.To4())

	// make a byte slice for the header
	// buf := make([]byte, header.TotalLen)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, header)

	header.Checksum = htons(checksum(buf.Bytes()))

	// remake it with the checksum also
	buf.Reset()
	binary.Write(buf, binary.BigEndian, header)

	return buf.Bytes(), nil
}
