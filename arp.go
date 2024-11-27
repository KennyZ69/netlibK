package main

import (
	"fmt"
	"net"
	"net/netip"
)

func (c *Client) ARPRequest(ip netip.Addr) error {
	if !c.SourceIp.IsValid() {
		return fmt.Errorf("Error invalid ip address of a client: %v\n")
	}

	arp, err := BuildARPPacket(OperationRequest, c.SourceIp, ip, c.SourceHardwareAddr, EthernetBroadcast)
	if err != nil {
		return err
	}

	return c.WriteTo(arp, EthernetBroadcast)
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
