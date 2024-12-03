package netlibk

import (
	"errors"
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

var (
	// Errors
	ErrInvalidClient = errors.New("Error invalid client source ip address")
	ErrInvalidIP     = errors.New("Error invalid ip address given")
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
	Version        uint8  // in Version also the IHL (HeaderLen) can be done (4 bits + 4 bits)
	HeaderLen      uint8  // IHL
	Service        uint8  // DSCP/ECN
	TotalLen       uint16 // total packet length (65535 bytes = max)
	Id             uint16
	Flags          uint16
	FragmentOffset uint16 // fragmentation info (could come together with flags)
	TTL            uint8  // time to live
	Protocol       uint8
	Checksum       uint16
	// SourceIp       netip.Addr
	// DestIp         netip.Addr
	SourceIp [4]byte
	DestIp   [4]byte
	// Options        uint32
}

type ICMPPacket struct {
	Type     uint8  // 1 byte
	Code     uint8  // 1 byte
	Checksum uint16 // 2 bytes
	Id       uint16 // 2 bytes
	Seq      uint16 // 2 bytes
	Payload  []byte // N bytes
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
	IpAddr       netip.Addr
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

func checksum(data []byte) uint16 {
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
	return uint16(^sum)
	// return uint16(^sum)
}
