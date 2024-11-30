package netlibk

import (
	"net"
	"net/netip"
	"syscall"
	"time"
)

type EtherType uint16
type Type int

const (
	// possible ethernet types
	ARP_PROTOCOL  EtherType = 0x0806
	IPv4_PROTOCOL EtherType = 0x0800
	IPv6_PROTOCOL EtherType = 0x086DD

	_ Type = iota
	SockRaw
	SockDatagram
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

var _ net.PacketConn = &RawConn{}

func (rc *RawConn) Close() error {
	return syscall.Close(rc.fd)
}

func (rc *RawConn) LocalAddr() net.Addr {
	return rc.localAddr
}

// read a packet from connection
func (rc *RawConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, _, err := syscall.Recvfrom(rc.fd, b, 0)
	if err != nil {
		return 0, nil, err
	}
	return n, rc.localAddr, err
}

// send a packet through the raw connection
func (rc *RawConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	n, err := syscall.Write(rc.fd, b)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (rc *RawConn) SetDeadline(t time.Time) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.readDeadline = t
	rc.writeDeadline = t
	return nil
}

func (rc *RawConn) SetReadDeadline(t time.Time) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.readDeadline = t
	return nil

}

func (rc *RawConn) SetWriteDeadline(t time.Time) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.writeDeadline = t
	return nil

}
