package netlibk

import (
	"net"
	"net/netip"
)

type EtherType uint16

const (
	// possible ethernet types
	ARP_PROTOCOL  EtherType = 0x806
	IPv4_PROTOCOL EtherType = 0x800
	IPv6_PROTOCOL EtherType = 0x86DD
)

type EthernetHeader struct {
	DestAddr net.HardwareAddr // 6 bytes, transmitted as-is
	// source hardware address for the frame (ethernet)
	SourceAddr net.HardwareAddr // 6 bytes, transmitted as-is
	EtherType  EtherType
	Payload    []byte
}

type Frame interface {
	net.HardwareAddr
	net.HardwareAddr
	EtherType
}

type IPv4Header struct {
	HeaderLen uint8
	TotalLen  uint16
	SourceIp  netip.Addr
	DestIp    netip.Addr
	Checksum  uint16
	Protocol  uint8
	TTL       uint8
	Id        uint16
	Flags     uint16
}

type ICMPPacket struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	Id       uint16
	Seq      uint16
	Payload  []byte
}

type ARPPacket struct {
	HardwareType       uint16    // 2 bytes
	ProtocolType       uint16    // 2 bytes
	HardwareAddrLength uint8     // 1 byte
	ProtocolLength     uint8     // 1 byte
	Operation          Operation // 2 bytes
	SenderHardwareAddr net.HardwareAddr
	SenderIp           netip.Addr
	TargetHardwareAddr net.HardwareAddr
	TargetIp           netip.Addr
}

// just to specify the operation as either reply or request
type Operation uint16

const (
	OperationRequest Operation = 1
	OperationReply   Operation = 2
)

type Address struct {
	HardwareAddr net.HardwareAddr
}

// this is now missing network and string method to implement the net.Addr inteface
var _ net.Addr = &Address{}

// return the network name for the address
func (adr *Address) Network() string {
	return "network"
}

func (adr *Address) String() string {
	return adr.HardwareAddr.String()
}
