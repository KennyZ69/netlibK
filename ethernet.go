package netlibk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

// a custom implementation of net.PacketConn
type RawConn struct {
	fd            int
	localAddr     net.Addr
	mu            sync.Mutex // to protect deadlines
	readDeadline  time.Time
	writeDeadline time.Time
}

// Broadcast is a hardware address of a frame that should be sent to every device on given subnet
var EthernetBroadcast = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

const minPayloadLen = 46

// build the headers, now with no error check but I may do it later idk
func BuildEthernetHeader(sourceMAC, destMAC net.HardwareAddr, etherType EtherType) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.Write(sourceMAC)
	buf.Write(destMAC)
	binary.Write(buf, binary.BigEndian, etherType)
	return buf.Bytes(), nil
}

// get the number of bytes required for this frame / eth header
func (et *EthernetHeader) length() int {
	pLen := len(et.Payload)
	// make the payload at least the mininal payload length so zero-pad it up till the min required
	if pLen < minPayloadLen {
		pLen = minPayloadLen
	}

	// 6 bytes for destination mac addr
	// 6 bytes for source mac addr
	// 2 bytes for ether type
	// N bytes for payload length (possibly padded)
	return 6 + 6 + 2 + pLen
}

// allocate a byte slice into the eth header / frame and make it to binary implementing the read func
func (et *EthernetHeader) Marshal() ([]byte, error) {
	// fmt.Printf("Length of the frame: %v\n", et.length())
	b := make([]byte, et.length())
	_, err := et.read(b)
	return b, err
}

// unmarshal byte slice into the ethernet header / frame
func (et *EthernetHeader) Unmarshal(b []byte) error {
	// 6 + 6 + 2 is the minimal size of the byte slice and then the payload length
	if len(b) < 14 {
		// return fmt.Errorf("Error byte slice of smaller size than 14")
		return io.ErrUnexpectedEOF
	}

	n := 14
	ethType := EtherType(binary.BigEndian.Uint16(b[n-2 : n]))
	et.EtherType = ethType

	// now make a byte slice for the mac dest and source and the payload (mac + mac + payload = length)
	bb := make([]byte, 6+6+len(b[n:]))
	copy(bb[0:6], b[0:6])
	et.DestAddr = bb[0:6]
	copy(bb[6:12], b[6:12])
	et.SourceAddr = bb[6:12]

	copy(bb[12:], b[12:])
	et.Payload = bb[12:]

	return nil
}

// make the binary form of a frame or eth header for the marshal to then allocate
func (et *EthernetHeader) read(b []byte) (int, error) {
	copy(b[0:6], et.DestAddr)
	copy(b[6:12], et.SourceAddr)
	n := 12

	// for now I do not care about VLAN I guess I do not need it for goapt

	binary.BigEndian.PutUint16(b[n:n+2], uint16(et.EtherType))
	// fmt.Printf("Copying payload: %v\n", et.Payload)
	copy(b[n+2:], et.Payload)
	return len(b), nil
}

// func Listen(ifi *net.Interface, socketType Type, protocol int) (net.PacketConn, error) {
func Listen(ifi *net.Interface, socketType Type, protocol int) (*RawConn, error) {
	// fmt.Printf("Protocol: 0x%04x, SocketType: %d\n", protocol, socketType)
	var fd int
	var err error

	// create socket

	// switch protocol {
	// case syscall.IPPROTO_ICMP:
	// 	fd, err = syscall.Socket(syscall.AF_INET, int(socketType), int(htons(uint16(protocol))))
	// case int(ARP_PROTOCOL):
	// 	fd, err = syscall.Socket(syscall.AF_PACKET, int(socketType), int(htons(uint16(protocol))))
	// }
	fd, err = syscall.Socket(syscall.AF_PACKET, int(socketType), int(htons(uint16(protocol))))
	if err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to create socket: %v\n", err)
	}

	// bind the socket
	sa := &syscall.SockaddrLinklayer{
		Protocol: htons(uint16(protocol)),
		Ifindex:  ifi.Index,
	}
	err = syscall.Bind(fd, sa)
	if err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("Error failed to bind socket: %v\n", err)
	}

	// // convert the socket into net.packetconn
	// f := os.NewFile(uintptr(fd), fmt.Sprintf("fd %v", fd))
	// fmt.Println("Got the os.File for the socket to convert into packetconn")
	// conn, err := net.FilePacketConn(f)
	// f.Close()
	// if err != nil {
	// 	return nil, fmt.Errorf("Error failed to create packet connection in Listen func: %v\n", err)
	// }
	// fmt.Println("Converted socket into packetconn successfully")

	// return conn, nil

	// missed error check
	addrs, _ := ifi.Addrs()
	// if err != nil {
	// }

	// missed error check
	addr, _ := getIPv4Addr(addrs)
	s := addr.AsSlice()

	return &RawConn{
		fd: fd,
		// localAddr: &net.IPAddr{
		// IP: net.IPv4zero,
		// },
		localAddr: &net.IPAddr{
			IP: net.IP(s),
		},
	}, nil
}

// htons converts a 16-bit value to big-endian
func htons(val uint16) uint16 {
	return (val<<8)&0xff00 | (val>>8)&0x00ff
}

// listen func for linux system
//	func listen(ifi *net.Interface, socketType Type, protocol int) (*net.PacketConn, error) {
//		var sockt int
//		switch socketType {
//		case SockRaw:
//			sockt = unix.SOCK_RAW
//		case SockDatagram:
//			sockt = unix.SOCK_DGRAM
//		default:
//			return nil, fmt.Errorf("Invalid packet type val")
//		}
//	}
//
