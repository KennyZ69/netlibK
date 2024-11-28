package main

import (
	"flag"
	"log"
	"net"
	"time"
)

var (
	// set a network interface for the tool
	ifiFlag = flag.String("i", "eth0", "network interface to use for the scanner")

	// set the timeout for the tool
	timeFlag = flag.Duration("d", 1*time.Second, "timeout to send the arp requests")
)

func main() {
	flag.Parse()

	// validate the network interface
	ifi, err := net.InterfaceByName(*ifiFlag)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: set the client for icmp and arp requests

	c, err := SetClient(ifi)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err = c.Conn.SetDeadline(time.Now().Add(time.Second * 2)); err != nil {
		log.Fatal(err)
	}

	// So now I have a client that can resolve ip addr to its source hardware addr -> mac addr
	// or so I am working on the resolving and retrieving
}
