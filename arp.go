package main

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
		HardwareType:       1,                     // default to 1 -> ethernet
		ProtocolType:       uint16(IPv4_PROTOCOL), // default to 0x800 ethernet type -> IPv4
		HardwareAddrLength: uint8(len(sourceMac)),
		ProtocolLength:     uint8(4),
		Operation:          op,
		SenderHardwareAddr: sourceMac,
		SenderIp:           sourceIp,
		TargetHardwareAddr: destMac,
		TargetIp:           targetIp,
	}, nil
}

func (p *ARPPacket) Marshal() ([]byte, error)

// Unmarshal a byte slice into arp packet struct
func (p *ARPPacket) Unmarshal(b []byte) error {
	if len(b) < 8 {
		return io.ErrUnexpectedEOF
	}

	p.HardwareType = binary.BigEndian.Uint16(b[0:2])
	p.ProtocolType = binary.BigEndian.Uint16(b[2:4])

	p.HardwareAddrLength = b[4]
	p.ProtocolLength = b[5]

	p.Operation = Operation(binary.BigEndian.Uint16(b[6:8]))

	n := 8

	// get the lengths times two because there is the sender and destination fields
	hlen := int(p.HardwareAddrLength)
	hlen2 := hlen * 2
	plen := int(p.ProtocolLength)
	plen2 := plen * 2

	arplen := n + plen + hlen
	if len(b) < arplen {
		return io.ErrUnexpectedEOF
	}

	bb := make([]byte, arplen-n)

	copy(bb[0:hlen], b[n:n+hlen])
	p.SenderHardwareAddr = bb[0:hlen]
	n += hlen

	copy(bb[hlen:hlen+plen], b[n:n+plen])
	senderIp, ok := netip.AddrFromSlice(bb[hlen : hlen+plen])
	if !ok {
		return fmt.Errorf("Invalid sender ip addr")
	}
	p.SenderIp = senderIp
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

	if fr.EtherType != ARP_PROTOCOL {
		return nil, nil, fmt.Errorf("Invalid ARP packet")
	}

	p := new(ARPPacket)
	err = p.Unmarshal(b)
	if err != nil {
		return nil, nil, err
	}

	return p, fr, nil
}
