package netlibk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// ping for when the base Ping function does not work (this is not working with raw sockets, but the standard library)
// or when you don't want to use client, so you want a higher level implementation (it is essentially easier to set up and run etc...)
func HigherLvlPing(dest net.IP, payload []byte, timeout time.Duration) (time.Duration, bool, error) {
	var seqN, icmpID uint16 = 1, uint16(os.Getpid() & 0xffff)
	if dest == nil || dest.To4() == nil {
		return 0, false, fmt.Errorf("invalid IP address")
	}
	fmt.Printf("Connecting to %s\n", dest.String())

	c, err := net.Dial("ip4:icmp", dest.String())
	if err != nil {
		return 0, false, fmt.Errorf("could not connect: %v", err)
	}
	defer c.Close()

	icmp := &ICMP{
		Type: 8,
		Code: 0,
		Id:   icmpID,
		Seq:  seqN,
	}
	fmt.Printf("Sending to %s: id=%v; seqn=%v\n", dest.String(), icmp.Id, icmp.Seq)

	packet := new(bytes.Buffer)
	if err := binary.Write(packet, binary.BigEndian, icmp); err != nil {
		return 0, false, fmt.Errorf("Error writing ICMP header: %v", err)
	}
	packet.Write(payload)
	icmp.checksum(packet.Bytes())

	packet.Reset()
	if err := binary.Write(packet, binary.BigEndian, icmp); err != nil {
		return 0, false, fmt.Errorf("Error writing ICMP header: %v", err)
	}
	packet.Write(payload)

	start := time.Now()
	if _, err := c.Write(packet.Bytes()); err != nil {
		return 0, false, fmt.Errorf("Error sending packet: %v", err)
	}

	reply := make([]byte, 1024)
	c.SetReadDeadline(time.Now().Add(timeout))
	n, err := c.Read(reply)
	if err != nil {
		return 0, false, fmt.Errorf("Error reading reply: %v", err)
	}

	duration := time.Since(start)

	if n < 28 { // Minimum length for an IPv4 + ICMP reply
		return 0, false, fmt.Errorf("invalid reply length")
	}

	replyID := binary.BigEndian.Uint16(reply[24:26])
	replySeq := binary.BigEndian.Uint16(reply[26:28])
	fmt.Printf("Received reply from %s: id=%v, seq=%v\n", dest.String(), replyID, replySeq)

	if replyID != icmp.Id || replySeq != icmp.Seq {
		return 0, false, fmt.Errorf("Error mismatched reply id or sequence number")
	}

	return duration, true, nil
}

// ping the desired destination ip with payload and return the response time, active boolean and error
func (c *Client) Ping(dest net.IP, payload []byte) (time.Duration, bool, error) {
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
	b := make([]byte, 128)
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

	fmt.Printf("Parsed ICMP: Type=%d, Code=%d, ID=%d, Seq=%d\n", icmp.Type, icmp.Code, icmp.Id, icmp.Seq)

	if len(b) > 8 {
		icmp.Payload = make([]byte, len(b)-8)
		copy(b[8:], icmp.Payload)
	} else {
		icmp.Payload = nil
	}

	return nil
}

func (c *Client) SendICMP(dest net.IP, payload []byte) error {
	if c.SourceIp == nil {
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

	// fmt.Printf("Raw Packet: %x\n", p)

	// sockaddr := &syscall.SockaddrInet4{}
	// copy(sockaddr.Addr[:], dest.To4())

	_, err = c.Conn.WriteTo(p, &net.IPAddr{IP: dest})
	if err != nil {
		return fmt.Errorf("Failed to send raw ICMP packet: %v\n", err)
	}

	// IF it doesn't work this way I can try to use the raw header and kernel configurations
	// and all the syscall things

	// fmt.Printf("Sending to IP: %v, sockaddr: %+v\n", dest, sockaddr)

	// fd := int(c.Conn.(*RawConn).fd)
	//
	// syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1)
	//
	// ipv4header, err := BuildIPv4Header(c.SourceIp, dest, uint16(IPv4_PROTOCOL), p)
	// if err != nil {
	// 	return fmt.Errorf("Error building IPv4 header for ping function: %v\n", err)
	// }
	// packet := append(ipv4header, p...)
	//
	// err = syscall.Sendto(fd, packet, 0, sockaddr)
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
	fmt.Printf("ID: %v : %v\nTYPE: %v\n", icmp.Id, c.ICMP_ID, icmp.Type)
	// if icmp.Id != c.ICMP_ID || icmp.Type != 0 {
	// 	return nil, 0, false, fmt.Errorf("Error unexpected icmp response packet\n")
	// }

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

func (icmp *ICMP) checksum(data []byte) {
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
