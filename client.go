package netlibk

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"syscall"
)

type Client struct {
	Iface              *net.Interface
	Conn               net.PacketConn
	SourceIp           netip.Addr
	SourceHardwareAddr net.HardwareAddr
	EthernetHeader     *EthernetHeader
	IPv4Header         *IPv4Header

	ICMP_ID    uint16
	ICMPSeqNum uint16
}

// func ICMPSetClientWhenInvalid(ifi *net.Interface, ip netip.Addr) (*Client, error) {
// 	conn, err := net.ListenPacket("ip4:icmp", ip.String())
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return New(ifi, conn)
// }

func ICMPSetClient(ifi *net.Interface) (*Client, error) {
	conn, err := Listen(ifi, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		return nil, fmt.Errorf("Error opening connection for the net interface: %v\n", err)
	}
	return New(ifi, conn)

}

func ARPSetClient(ifi *net.Interface) (*Client, error) {
	// for now using the "ethernet" but I want to have something for non ethernet also
	// I found that it probably won't work through wifi
	// conn, err := net.ListenPacket("ethernet", ifi.Name)
	conn, err := Listen(ifi, syscall.SOCK_RAW, int(ARP_PROTOCOL))
	if err != nil {
		return nil, fmt.Errorf("Error opening connection for the net interface: %v\n", err)
	}
	return New(ifi, conn)
}

// create a new client using the network interface and packet connection
// it would be better to just use the SetClient function
func New(ifi *net.Interface, conn net.PacketConn) (*Client, error) {
	// TODO: build the eth and ipv headers and get the client details -> ip and mac addr to assign

	// get the usable IPv4 for the user on his network interface
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, fmt.Errorf("Error getting the IPv4 address for the user: %v\n", err)
	}

	// ipAddrs := make([]netip.Addr, len(addrs))
	// for i, addr := range addrs {
	// 	ipPrefix, err := netip.ParsePrefix(addr.String())
	// 	if err != nil {
	// 		return nil, fmt.Errorf("Erorr parsing prefix of an addr: %v\n", err)
	// 	}
	// 	ipAddrs[i] = ipPrefix.Addr()
	// }

	// return newClient(ifi, conn, ipAddrs)
	return newClient(ifi, conn, addrs)
}

// func newClient(ifi *net.Interface, conn net.PacketConn, addrs []netip.Addr) (*Client, error) {
func newClient(ifi *net.Interface, conn net.PacketConn, addrs []net.Addr) (*Client, error) {
	ip, err := getIPv4Addr(addrs)
	if err != nil {
		return nil, err
	}

	sourceMac := ifi.HardwareAddr

	// BuildEthernetHeader(sourceMac, )

	return &Client{
		Iface:              ifi,
		Conn:               conn,
		SourceIp:           ip,
		SourceHardwareAddr: sourceMac,

		// unique id based on process id
		ICMP_ID:    uint16(os.Getpid() & 0xffff),
		ICMPSeqNum: 1,
	}, nil
}

func (c *Client) Close() error {
	return c.Conn.Close()
}

// func getIPv4Addr(addrs []netip.Addr) (netip.Addr, error) {
func getIPv4Addr(addrs []net.Addr) (netip.Addr, error) {
	ipAddrs := make([]netip.Addr, len(addrs))

	for i, addr := range addrs {
		ipPrefix, err := netip.ParsePrefix(addr.String())
		if err != nil {
			return netip.Addr{}, fmt.Errorf("Erorr parsing prefix of an addr: %v\n", err)
		}
		ipAddrs[i] = ipPrefix.Addr()
	}
	for _, addr := range ipAddrs {
		if addr.Is4() {
			return addr, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("No valid IPv4 address")
}

func (c *Client) HardwareAddr() net.HardwareAddr {
	return c.Iface.HardwareAddr
}

func (c *Client) Write(p *ARPPacket, addr net.HardwareAddr) error {
	payload, err := p.Marshal()
	if err != nil {
		return err
	}

	et := &EthernetHeader{
		DestAddr:   addr,                 // 6 bytes
		SourceAddr: c.SourceHardwareAddr, // 6 bytes
		EtherType:  ARP_PROTOCOL,         // 2 bytes
		Payload:    payload,              // N bytes
	}

	// I guess I need to implement reading the data from the struct into bytes
	b, err := et.Marshal()
	if err != nil {
		return err
	}

	// because I want to write the payload to the address I need to first make the payload by marshalling
	_, err = c.Conn.WriteTo(b, &Address{HardwareAddr: addr})
	return err
}

func (c *Client) ResolveMAC(targetIp netip.Addr) (net.HardwareAddr, error) {
	err := c.ARPRequest(targetIp)
	if err != nil {
		return nil, err
	}
	fmt.Println("Sent the request")

	// wait and get the replies
	for {
		fmt.Println("Receiving the reply")
		arp, _, err := c.ReceiveARP()
		if err != nil {
			return nil, err
		}
		fmt.Println("Reply received")

		fmt.Printf("Sender ip: %v; Target ip: %v\nOp: %v\n", arp.SenderIp, targetIp, arp.Operation)
		// if arp.Operation != OperationReply || arp.SenderIp != targetIp {
		if arp.SenderIp != targetIp {
			continue
		}

		return arp.SenderHardwareAddr, nil
	}
}
