package main

import (
	"fmt"
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

func (p *ARPPacket) Unmarshal([]byte) error

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
