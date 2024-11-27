package main

import (
	"fmt"
	"net"
	"net/netip"
)

type Client struct {
	Iface              *net.Interface
	Conn               net.PacketConn
	SourceIp           netip.Addr
	SourceHardwareAddr net.HardwareAddr
	EthernetHeader     *EthernetHeader
	IPv4Header         *IPv4Header
}

func SetClient(ifi *net.Interface) (*Client, error) {
	// for now using the "ethernet" but I want to have something for non ethernet also
	conn, err := net.ListenPacket("ethernet", ifi.Name)
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

	ipAddrs := make([]netip.Addr, len(addrs))
	for i, addr := range addrs {
		ipPrefix, err := netip.ParsePrefix(addr.String())
		if err != nil {
			return nil, fmt.Errorf("Erorr parsing prefix of an addr: %v\n", err)
		}
		ipAddrs[i] = ipPrefix.Addr()
	}

	return newClient(ifi, conn, ipAddrs)
}

func newClient(ifi *net.Interface, conn net.PacketConn, addrs []netip.Addr) (*Client, error) {
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
	}, nil
}

func (c *Client) Close() error {
	return c.Conn.Close()
}

func getIPv4Addr(addrs []netip.Addr) (netip.Addr, error) {
	for _, addr := range addrs {
		if addr.Is4() {
			return addr, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("No valid IPv4 address")
}

func (c *Client) HardwareAddr() net.HardwareAddr {
	return c.Iface.HardwareAddr
}

type Address struct {
	HardwareAddr net.HardwareAddr
}

// I am thinking I could do this with generics because I need to use it with arp and icmp too
func (c *Client) Write(p *ARPPacket, addr net.HardwareAddr) error {
	//
	payload, err := p.Marshal()
	if err != nil {
		return err
	}

	st := &struct {
		DestMac      net.HardwareAddr
		SourceMac    net.HardwareAddr
		EthernetType EtherType
		Payload      []byte
	}{
		DestMac:      addr,
		SourceMac:    c.SourceHardwareAddr,
		EthernetType: ARP_PROTOCOL,
		Payload:      payload,
	}

	// I guess I need to implement reading the data from the struct into bytes
	b, blen, err := read(st)

	// because I want to write the payload to the address I need to first make the payload by marshalling
	_, err = c.Conn.WriteTo(stBin, &Address{HardwareAddr: addr})
	return err
}

func read[st struct {
	DestMac      net.HardwareAddr
	SourceMac    net.HardwareAddr
	EthernetType EtherType
	Payload      []byte
}](*st) ([]byte, int, error) {

}
