package netlibk

func (icmp *ICMPPacket) checksum(data []byte) {
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
	icmp.Checksum = uint16(^sum)
	// return uint16(^sum)
}
