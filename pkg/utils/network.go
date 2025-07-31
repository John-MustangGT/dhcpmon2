// ===== pkg/utils/network.go =====
package utils

import (
	"encoding/binary"
	"net"
)

// IPToInt converts an IP address to a 32-bit integer for sorting
func IPToInt(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// IntToIP converts a 32-bit integer back to an IP address
func IntToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

// IsPrivateMAC checks if a MAC address is a locally administered (private) MAC
func IsPrivateMAC(mac net.HardwareAddr) bool {
	if len(mac) == 0 {
		return false
	}
	// Check if the locally administered bit (bit 1 of the first octet) is set
	return (mac[0] & 0x02) != 0
}

// NormalizeMAC normalizes a MAC address string to uppercase with colons
func NormalizeMAC(mac string) string {
	if hwAddr, err := net.ParseMAC(mac); err == nil {
		return hwAddr.String()
	}
	return mac
}

