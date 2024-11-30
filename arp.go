package netlibk

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/netip"
)

func (c *Client) ARPRequest(ip netip.Addr) error {
	if !c.SourceIp.IsValid() {
		return fmt.Errorf("Error invalid ip address of a client\n")
	}

	arp, err := BuildARPPacket(OperationRequest, c.SourceIp, ip, c.SourceHardwareAddr, EthernetBroadcast)
	if err != nil {
		return err
	}

	// ethFrameB, err := BuildEthernetHeader()
	// if err != nil {
	// 	return fmt.Errorf("Error building ethernet header: %v\n", err)
	// }

	return c.Write(arp, EthernetBroadcast)
}

func BuildARPPacket(op Operation, sourceIp, targetIp netip.Addr, sourceMac, destMac net.HardwareAddr) (*ARPPacket, error) {

	return &ARPPacket{
		HardwareType: 1,                     // default to 1 -> ethernet
		ProtocolType: uint16(IPv4_PROTOCOL), // default to 0x800 ethernet type -> IPv4
		// HardwareAddrLength: uint8(len(sourceMac)),
		HardwareAddrLength: uint8(4),
		ProtocolLength:     uint8(4),
		Operation:          op,
		SenderHardwareAddr: sourceMac,
		SenderIp:           sourceIp,
		TargetHardwareAddr: destMac,
		TargetIp:           targetIp,
	}, nil
}

// malloc a byte slice with the packet details
func (p *ARPPacket) Marshal() ([]byte, error) {
	// 2 bytes for HardwareType
	// 2 bytes for ProtocolType
	// 1 byte for HardwareAddrLength
	// 1 byte for ProtocolLength
	// 2 bytes for Operation
	// N bytes for SenderHardwareAddr
	// N bytes for SenderIp
	// N bytes for TargetHardwareAddr
	// N bytes for TargetIp

	b := make([]byte, 2+2+1+1+2+(p.HardwareAddrLength*2)+(p.ProtocolLength*2))

	binary.BigEndian.PutUint16(b[0:2], p.HardwareType)
	binary.BigEndian.PutUint16(b[2:4], p.ProtocolType)

	b[4] = p.HardwareAddrLength
	b[5] = p.ProtocolLength
	fmt.Printf("Marshalled the p lenght: %v; and mac length: %v --> %v : %v\n", p.ProtocolLength, p.HardwareAddrLength, b[4], b[5])

	binary.BigEndian.PutUint16(b[6:8], uint16(p.Operation))

	n := 8
	hlen := int(p.HardwareAddrLength)
	plen := int(p.ProtocolLength)

	copy(b[n:n+hlen], p.SenderHardwareAddr)
	n += hlen

	senderIp := p.SenderIp.As4()
	// 8 + hardware length to the same + protocol length
	copy(b[n:n+plen], senderIp[:])
	fmt.Printf("Sender ip: %v\n", senderIp)
	n += plen

	copy(b[n:n+hlen], p.TargetHardwareAddr)
	n += hlen

	targetIp := p.TargetIp.As4()
	copy(b[n:n+plen], targetIp[:])
	fmt.Printf("Target ip: %v\n", targetIp)

	return b, nil
}

// Unmarshal a byte slice into arp packet struct
func (p *ARPPacket) Unmarshal(b []byte) error {
	if len(b) < 8 {
		return io.ErrUnexpectedEOF
	}

	p.HardwareType = binary.BigEndian.Uint16(b[0:2])
	p.ProtocolType = binary.BigEndian.Uint16(b[2:4])

	// p.HardwareAddrLength = b[4]
	fmt.Println("Warning: invalid hardware addr length, defaulting to 6")
	p.HardwareAddrLength = uint8(6)
	if p.HardwareAddrLength == 0 {
		fmt.Println("Warning: invalid hardware addr length, defaulting to 6")
		p.HardwareAddrLength = uint8(6)
	}
	p.ProtocolLength = b[5]
	if p.ProtocolLength == 0 {
		fmt.Println("Warning: invalid ip length, defaulting to 4")
		p.ProtocolLength = uint8(4)
	}

	fmt.Printf("HardwareAddrLength: %d, ProtocolLength: %d\n", p.HardwareAddrLength, p.ProtocolLength)

	p.Operation = Operation(binary.BigEndian.Uint16(b[6:8]))

	n := 8

	// get the lengths times two because there is the sender and destination fields
	hlen := int(p.HardwareAddrLength)
	hlen2 := hlen * 2
	plen := int(p.ProtocolLength)
	plen2 := plen * 2

	// to retrieve both mac and ip addresses
	arplen := n + plen2 + hlen2
	if len(b) < arplen {
		return io.ErrUnexpectedEOF
	}

	fmt.Printf("ARPLen: %d, Total Bytes: %d\n", arplen, len(b))

	bb := make([]byte, arplen-n)

	// sender mac
	copy(bb[0:hlen], b[n:n+hlen])
	p.SenderHardwareAddr = bb[0:hlen]
	n += hlen

	// sender ip
	copy(bb[hlen:hlen+plen], b[n:n+plen])
	senderIp := b[n : n+plen]
	fmt.Printf("Sender IP bytes: %x\n", senderIp)
	ip, ok := netip.AddrFromSlice(senderIp)
	// senderIp, ok := netip.AddrFromSlice(bb[hlen : hlen+plen])
	if !ok {
		return fmt.Errorf("Invalid sender ip addr: %x", senderIp)
	}
	// p.SenderIp = senderIp
	p.SenderIp = ip
	n += plen

	copy(bb[hlen+plen:hlen2+plen], b[n:n+plen])
	p.TargetHardwareAddr = bb[hlen+plen : hlen2+plen]
	n += plen

	copy(bb[hlen2+plen:hlen2+plen2], b[n:n+plen])
	targetIP, ok := netip.AddrFromSlice(bb[hlen2+plen : hlen2+plen2])
	if !ok {
		return fmt.Errorf("Invalid target ip addr")
	}
	p.TargetIp = targetIP

	return nil
}

func parsePacket(b []byte) (*ARPPacket, *EthernetHeader, error) {
	fr := new(EthernetHeader)
	err := fr.Unmarshal(b)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("Unmarshalled the frame")

	if fr.EtherType != ARP_PROTOCOL {
		return nil, nil, fmt.Errorf("Invalid ARP packet")
	}

	// unmarshal the sent payload into the new packet
	p := new(ARPPacket)
	err = p.Unmarshal(fr.Payload)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("Unmarshalled the packet")

	return p, fr, nil
}
