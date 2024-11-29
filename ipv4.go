package netlibk

import "net/netip"

func BuildIPv4Header(sourceIp, destIp netip.Addr, protocol uint8, payloadlen uint16) ([]byte, error)
