package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"time"

	"github.com/KennyZ69/netlibK"
)

var (
	// set a network interface for the tool (for me "eno1" is the net interface)
	ifiFlag = flag.String("i", "eno1", "network interface to use for the scanner")

	// set the timeout for the tool
	timeFlag = flag.Duration("d", 2*time.Second, "timeout to send the arp requests")

	// ip flag for test purposes, in the goapt tool I will have already gotten possible ips
	ipFlag = flag.String("ip", "", "IPv4 target address to send the arp request to")
)

func main() {
	flag.Parse()

	// validate the network interface
	fmt.Println("Getting the net interface")
	ifi, err := net.InterfaceByName(*ifiFlag)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(ifi)

	// TODO: set the client for icmp and arp requests

	fmt.Println("Setting up the client")
	c, err := netlibk.SetClient(ifi)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	fmt.Println("Client running")

	if err = c.Conn.SetDeadline(time.Now().Add(*timeFlag)); err != nil {
		log.Fatal(err)
	}

	// So now I have a client that can resolve ip addr to its source hardware addr -> mac addr
	// or so I am working on the resolving and retrieving

	fmt.Println("Parsing the ip address provided")
	ip, err := netip.ParseAddr(*ipFlag)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Resolving the mac address to the given ip")
	mac, err := c.ResolveMAC(ip)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("ip: %s --> mac: %s\n", ip, mac)
}
