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
	conn, err := net.ListenPacket("ethernet", ifi.Name)
	if err != nil {
		return nil, fmt.Errorf("Error open connection for the net interface: %v\n", err)
	}
	return New(ifi, conn)
}

func New(ifi *net.Interface, conn net.PacketConn) (*Client, error) {
	// TODO: build the eth and ipv headers and get the client details -> ip and mac addr to assign
}

func (c *Client) Close() error {
	return c.Conn.Close()
}
