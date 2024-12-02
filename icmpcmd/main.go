package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"time"

	netlibk "github.com/KennyZ69/netlibK"
)

var (
	// set a network interface for the tool (for me "eno1" is the net interface)
	ifiFlag = flag.String("i", "eno1", "network interface to use for the scanner")

	// set the timeout for the tool
	timeFlag = flag.Duration("d", 2*time.Second, "timeout to send the arp requests")

	ipFlag = flag.String("ip", "", "Ip address to ping")
)

func main() {
	flag.Parse()
	ip, err := netip.ParseAddr(*ipFlag)
	if err != nil {
		log.Fatal(err)
	}

	dur, active, err := netlibk.HigherLvlPing(ip, []byte("Hello world!"), *timeFlag)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Ping to %s: %v: %v\n", ip, active, dur)

	ifi, err := net.InterfaceByName(*ifiFlag)
	if err != nil {
		log.Fatal(err)
	}

	c, err := netlibk.ICMPSetClient(ifi)
	// c, err := netlibk.ICMPSetClientWhenInvalid(ifi, ip)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err = c.Conn.SetDeadline(time.Now().Add(*timeFlag)); err != nil {
		log.Fatal(err)
	}

	t, active, err := c.Ping(ip, []byte("Hello world!"))
	if err != nil {
		log.Fatal(err)
	}

	if active {
		fmt.Printf("Ping to %s: %v; %v\n", ip, active, t)
	}
}
