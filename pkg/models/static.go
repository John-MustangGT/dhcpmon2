// ===== pkg/models/static.go =====
package models

import (
	"fmt"
	"net"
	"strings"
)

// StaticDHCPEntry represents a static DHCP reservation
type StaticDHCPEntry struct {
	ID          string           `json:"id"`          // Unique identifier
	MAC         net.HardwareAddr `json:"mac"`         // MAC address
	IP          net.IP           `json:"ip"`          // Assigned IP address
	Hostname    string           `json:"hostname"`    // Hostname
	Tag         string           `json:"tag"`         // Network tag (optional)
	LeaseTime   string           `json:"leaseTime"`   // Lease time (optional)
	Comment     string           `json:"comment"`     // Comment (optional)
	Enabled     bool             `json:"enabled"`     // Whether entry is enabled
	LineNumber  int              `json:"lineNumber"`  // Original line number in file
	RawLine     string           `json:"rawLine"`     // Original raw line
}

// ToDnsmasqLine converts the entry back to dnsmasq configuration format
func (e *StaticDHCPEntry) ToDnsmasqLine() string {
	if !e.Enabled {
		return "# " + e.RawLine
	}

	parts := []string{}
	
	// MAC address is required
	if e.MAC != nil {
		parts = append(parts, e.MAC.String())
	}
	
	// Add tag if specified
	if e.Tag != "" {
		parts = append(parts, "set:"+e.Tag)
	}
	
	// Add IP if specified
	if e.IP != nil {
		parts = append(parts, e.IP.String())
	}
	
	// Add hostname if specified
	if e.Hostname != "" {
		parts = append(parts, e.Hostname)
	}
	
	// Add lease time if specified
	if e.LeaseTime != "" {
		parts = append(parts, e.LeaseTime)
	}
	
	line := "dhcp-host=" + strings.Join(parts, ",")
	
	// Add comment if specified
	if e.Comment != "" {
		line += " # " + e.Comment
	}
	
	return line
}

// Validate checks if the entry has required fields
func (e *StaticDHCPEntry) Validate() error {
	if e.MAC == nil {
		return fmt.Errorf("MAC address is required")
	}
	if e.IP == nil && e.Hostname == "" {
		return fmt.Errorf("either IP address or hostname is required")
	}
	return nil
}

