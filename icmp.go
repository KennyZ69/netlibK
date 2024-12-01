package netlibk

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/netip"
	"time"
)

func (c *Client) Ping(dest netip.Addr, payload []byte) (time.Duration, bool, error) {
	err := c.SendICMP(dest, payload)
	if err != nil {
		return 0, false, err
	}

	_, t, active, err := c.ReceiveICMP()
	if err != nil {
		return 0, active, err
	}

	return t, active, nil
}

func (icmp *ICMPPacket) Marshal() ([]byte, error) {
	b := make([]byte, 1+1+2+2+2)
	b[0] = icmp.Type
	b[1] = icmp.Code
	binary.BigEndian.PutUint16(b[2:4], icmp.Checksum)
	binary.BigEndian.PutUint16(b[4:6], icmp.Id)
	binary.BigEndian.PutUint16(b[6:8], icmp.Seq)
	copy(b[8:], icmp.Payload)

	return b, nil
}

func (icmp *ICMPPacket) Unmarshal(b []byte) error {
	if len(b) < 8 {
		return io.ErrUnexpectedEOF
	}

	icmp.Type = b[0]
	icmp.Code = b[1]

	icmp.Checksum = binary.BigEndian.Uint16(b[2:4])
	icmp.Id = binary.BigEndian.Uint16(b[4:6])
	icmp.Seq = binary.BigEndian.Uint16(b[6:8])

	if len(b) > 8 {
		icmp.Payload = make([]byte, len(b)-8)
		copy(b[8:], icmp.Payload)
	}
	icmp.Payload = nil

	return nil
}

func (c *Client) SendICMP(dest netip.Addr, payload []byte) error {
	if !c.SourceIp.IsValid() {
		return ErrInvalidClient
	}
	icmp, err := BuildICMPPacket(c.ICMPSeqNum, c.ICMP_ID, payload)
	if err != nil {
		return err
	}
	icmp.checksum(payload)
	c.ICMPSeqNum++

	p, err := icmp.Marshal()
	if err != nil {
		return err
	}

	_, err = c.Conn.WriteTo(p, &net.IPAddr{IP: dest.AsSlice()})
	if err != nil {
		return fmt.Errorf("Failed to send raw ICMP packet: %v\n", err)
	}

	// // Create sockaddr
	// sockaddr := &syscall.SockaddrInet4{}
	// copy(sockaddr.Addr[:], dest.AsSlice())
	//
	// // Send packet using raw file descriptor
	// fd := int(c.Conn.(*RawConn).fd) // Assuming RawConn structure has fd field
	// err = syscall.Sendto(fd, p, 0, sockaddr)
	// if err != nil {
	// 	return fmt.Errorf("Failed to send raw ICMP packet: %v\n", err)
	// }

	return nil
}

func (c *Client) ReceiveICMP() (*ICMPPacket, time.Duration, bool, error) {
	buf := make([]byte, 128)

	start := time.Now()
	n, _, err := c.Conn.ReadFrom(buf)
	if err != nil {
		return nil, 0, false, fmt.Errorf("Error reading from buffer when receiving icmp packet: %v\n", err)
	}

	icmp := &ICMPPacket{}
	if err = icmp.Unmarshal(buf[:n]); err != nil {
		return nil, 0, false, fmt.Errorf("Error unmarshalling: %v\n", err)
	}

	// check whether ids are correct and the type is response (0)
	if icmp.Id != c.ICMP_ID || icmp.Type != 0 {
		return nil, 0, false, fmt.Errorf("Error unexpected icmp response packet\n")
	}

	return icmp, time.Since(start), true, nil
}

func BuildICMPPacket(seq, id uint16, payload []byte) (*ICMPPacket, error) {
	return &ICMPPacket{
		Type:    8,
		Code:    0,
		Id:      id,
		Seq:     seq,
		Payload: payload,
	}, nil
}

func (icmp *ICMPPacket) checksum(data []byte) {
	var sum uint32

	// converting, shifting the bits and the "|" is a bitwise OR to combine those two 8-bit values into one 16 bit val
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	// ensuring no overflown bits remain there, extracting them and adding them to the lower 16 bits
	sum = (sum >> 16) + (sum & 0xffff)
	sum += (sum >> 16)

	// one's complement -> inverts all bits so 0 to 1 and 1 to 0
	icmp.Checksum = uint16(^sum)
	// return uint16(^sum)
}
