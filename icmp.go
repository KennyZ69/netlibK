package netlibk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"time"
)

// ping for when the base Ping function does not work (this is not working with raw sockets, but the standard library)
// or when you don't want to use client, so you want a higher level imolementation
func HigherLvlPing(dest netip.Addr, payload []byte, timeout time.Duration) (time.Duration, bool, error) {
	var seqN, icmp_id uint16 = 1, uint16(os.Getpid() & 0xffff)
	if !dest.IsValid() {
		return 0, false, ErrInvalidIP
	}
	// c, err := net.Dial("ip4:icmp", dest.String())
	c, err := net.Dial("ip4:icmp", "192.168.104.229")
	if err != nil {
		return 0, false, fmt.Errorf("Error could not connect to ListenPacket: %v\n", err)
	}
	defer c.Close()

	icmp, err := BuildICMPPacket(seqN, icmp_id, payload)
	if err != nil {
		return 0, false, err
	}
	icmp.checksum(payload)
	seqN++

	fmt.Printf("Sending to %s: id=%v; seqn=%v\n", dest.String(), icmp.Id, icmp.Seq)

	// p, err := icmp.Marshal()
	// if err != nil {
	// 	return 0, false, err
	// }
	//
	// start := time.Now()
	// _, err = c.Write(p)
	// if err != nil {
	// 	return 0, false, err
	// }
	//
	// reply := make([]byte, 1024)
	// if err = c.SetDeadline(time.Now().Add(timeout)); err != nil {
	// 	return 0, false, fmt.Errorf("Error setting deadline on net.Conn: %v\n", err)
	// }
	//
	// n, err := c.Read(reply)
	// if err != nil {
	// 	return 0, false, fmt.Errorf("Error reading the reply from connection: %v\n", err)
	// }
	//
	// duration := time.Since(start)
	//
	// if n < 28 { // 20 + 8 bytes as the minimum length for ipv4 header + icmp packet
	// 	return 0, false, fmt.Errorf("Error invalid ICMP reply length")
	// }
	//
	// replyId := binary.BigEndian.Uint16(reply[24:26])
	// replySeq := binary.BigEndian.Uint16(reply[26:28])
	// if replyId != icmp.Id || replySeq != icmp.Seq {
	// 	return 0, false, fmt.Errorf("Error mismatched ICMP reply ID or Seq number")
	// }

	packet := new(bytes.Buffer)
	binary.Write(packet, binary.BigEndian, icmp)
	// payload = []byte("Incoming ping...")
	packet.Write(payload)
	// icmpPacket.Checksum = getChecksum(packet.Bytes())
	// set the checksum of the packet
	// icmpPacket.checksum(packet.Bytes())

	packet.Reset()

	binary.Write(packet, binary.BigEndian, icmp)
	packet.Write(payload)

	// send the icmp
	start := time.Now()
	pl, err := c.Write(packet.Bytes())
	if err != nil {
		return 0, false, fmt.Errorf("Error sending icmp packet: %v\n", err)
	}
	fmt.Printf("Length of packet written: %v\n", pl)

	// handle the reply
	reply := make([]byte, 1024)
	c.SetReadDeadline(time.Now().Add(timeout))
	n, err := c.Read(reply)
	if err != nil {
		return 0, false, fmt.Errorf("Error handling the icmp reply: %v\n", err)
	}
	fmt.Printf("Length of reply: %v\n", n)

	duration := time.Since(start)

	if n < 28 { // 20 + 8 as the minimum length for ipv4 header + icmp header
		return 0, false, fmt.Errorf("Error invalid ICMP reply lenght")
	}

	replyId := binary.BigEndian.Uint16(reply[23:25])
	replySeq := binary.BigEndian.Uint16(reply[25:27])
	// replyPayload := binary.BigEndian.Uint16(reply[28:])
	fmt.Printf("replyId=%v; replySeq=%v\n", replyId, replySeq)
	if replyId != icmp.Id || replySeq != icmp.Seq {
		return 0, false, fmt.Errorf("Error mismatched ICMP reply ID or Seq number")
	}

	return duration, true, nil
}

// ping the desired destination ip with payload and return the response time, active boolean and error
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
