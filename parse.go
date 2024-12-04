package netlibk

import (
	"fmt"
	"log"
	"net"
	"strings"
)

func ParseIPInputs(ips []string) (string, string, bool, error) {
	var addrStart, addrEnd string

	// CIDR notation
	if strings.Contains(ips[0], "/") {
		_, _, err := net.ParseCIDR(ips[0])
		if err != nil {
			return "", "", false, fmt.Errorf("Invalid CIDR\n")
		}
		// TODO: I have to somehow return the first and last ip from the CIDR range
		return ips[0], ips[0], true, nil
	}

	addrStart = ResolveHostname(ips[0])
	if addrStart == "" {
		return "", "", false, fmt.Errorf("Error resolving the hostname or a single IP")
	}

	if ips[1] == "" {
		addrEnd = addrStart
	} else {
		addrEnd = ResolveHostname(ips[1])
	}

	fmt.Printf("Have the start: %v; and end: %v\n", addrStart, addrEnd)

	return addrStart, addrEnd, false, nil
}

func ResolveHostname(input string) string {
	fmt.Println("Resolving hostname on ", input)
	ips, err := net.LookupIP(input)
	if err != nil || len(ips) == 0 {
		fmt.Println("Error looking up the host to resolve")
		return ""
	}
	return ips[0].String()
}

func GenerateIPs(startIP, endIP string) []net.IP {
	var ips []net.IP
	fmt.Printf("Generating from %v to %v\n", startIP, endIP)
	start := net.ParseIP(startIP)
	if start == nil {
		log.Fatalf("Error parsing start IP: %s\n", startIP)
	}
	end := net.ParseIP(endIP)
	if end == nil {
		log.Fatalf("Error parsing end IP: %s\n", endIP)
	}

	for ip := start; CompareIPs(ip, end) <= 0; {
		ips = append(ips, ip)

		var ok bool
		ip, ok = inc(ip)
		if !ok {
			log.Fatalf("Error incrementing IP %v\n", ip)
		}
	}
	log.Printf("Generated IPs to from %v to %v ... \n", startIP, endIP)

	return ips
}

func CompareIPs(ip1, ip2 net.IP) int {
	ip1 = ip1.To16()
	ip2 = ip2.To16()

	for i := 0; i < len(ip1) && i < len(ip2); i++ {
		if ip1[i] < ip2[i] {
			return -1
		} else if ip1[i] > ip2[i] {
			return 1
		}
	}

	return 0
}

func GenerateIPsFromCIDR(input string) []net.IP {
	_, ipNet, err := net.ParseCIDR(input)
	if err != nil {
		log.Fatalf("Error parsing CIDR: %v\n", err)
	}

	var ips []net.IP
	for ip := ipNet.IP; ipNet.Contains(ip); {
		ips = append(ips, ip)

		var ok bool
		ip, ok = inc(ip)
		if !ok {
			log.Fatalf("Error incrementing IP %v\n", ip)
		}
	}
	log.Printf("Generating IPs to scan from %v to %v ... \n", ips[0], ips[len(ips)-1])

	return ips
}

func inc(ip net.IP) (net.IP, bool) {
	ip = ip.To16()
	if ip == nil {
		log.Fatalf("Invalid IP for increment: %v\n", ip)
	}
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)

	fmt.Printf("Incrementing IP: %v\n", ip)

	// increment from the last byte
	for i := len(newIP) - 1; i >= 0; i-- {
		newIP[i]++
		if newIP[i] != 0 { // No overflow, stop here
			break
		}
	}

	// Check for overflow
	if newIP.Equal(net.IPv4zero) {
		return nil, false
	}
	return newIP, true
}
