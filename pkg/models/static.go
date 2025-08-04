// ===== pkg/models/static.go (Updated with proper MAC formatting) =====
package models

import (
	"encoding/json"
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

// MarshalJSON customizes JSON marshaling to format MAC address properly
func (e StaticDHCPEntry) MarshalJSON() ([]byte, error) {
	type Alias StaticDHCPEntry
	
	return json.Marshal(&struct {
		MAC string `json:"mac"`
		IP  string `json:"ip,omitempty"`
		*Alias
	}{
		MAC:   e.GetFormattedMAC(),
		IP:    e.GetFormattedIP(),
		Alias: (*Alias)(&e),
	})
}

// GetFormattedMAC returns MAC address in standard AA:BB:CC:DD:EE:FF format
func (e *StaticDHCPEntry) GetFormattedMAC() string {
	if e.MAC == nil {
		return ""
	}
	// Convert to uppercase and ensure colon format
	return strings.ToUpper(e.MAC.String())
}

// GetFormattedIP returns IP address as string, or empty if nil
func (e *StaticDHCPEntry) GetFormattedIP() string {
	if e.IP == nil {
		return ""
	}
	return e.IP.String()
}

// SetMAC parses and sets the MAC address from string, normalizing format
func (e *StaticDHCPEntry) SetMAC(macStr string) error {
	if macStr == "" {
		return fmt.Errorf("MAC address cannot be empty")
	}
	
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return fmt.Errorf("invalid MAC address format: %w", err)
	}
	
	e.MAC = mac
	return nil
}

// SetIP parses and sets the IP address from string
func (e *StaticDHCPEntry) SetIP(ipStr string) error {
	if ipStr == "" {
		e.IP = nil
		return nil
	}
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format: %s", ipStr)
	}
	
	e.IP = ip
	return nil
}

// ToDnsmasqLine converts the entry back to dnsmasq configuration format
func (e *StaticDHCPEntry) ToDnsmasqLine() string {
	if !e.Enabled {
		return "# " + e.RawLine
	}

	parts := []string{}
	
	// MAC address is required (formatted as AA:BB:CC:DD:EE:FF)
	if e.MAC != nil {
		parts = append(parts, e.GetFormattedMAC())
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

// Validate checks if the entry has required fields and valid formats
func (e *StaticDHCPEntry) Validate() error {
	if e.MAC == nil {
		return fmt.Errorf("MAC address is required")
	}
	
	// Validate MAC address format (should be 6 bytes)
	if len(e.MAC) != 6 {
		return fmt.Errorf("invalid MAC address length")
	}
	
	if e.IP == nil && e.Hostname == "" {
		return fmt.Errorf("either IP address or hostname is required")
	}
	
	// Validate IP address if provided
	if e.IP != nil {
		// Check if it's IPv4
		if e.IP.To4() == nil {
			return fmt.Errorf("only IPv4 addresses are supported")
		}
	}
	
	// Validate hostname format if provided
	if e.Hostname != "" {
		if len(e.Hostname) > 253 {
			return fmt.Errorf("hostname too long (max 253 characters)")
		}
		
		// Basic hostname validation (no spaces, special characters)
		for _, char := range e.Hostname {
			if !((char >= 'a' && char <= 'z') || 
				 (char >= 'A' && char <= 'Z') || 
				 (char >= '0' && char <= '9') || 
				 char == '-' || char == '.') {
				return fmt.Errorf("hostname contains invalid characters")
			}
		}
	}
	
	return nil
}

// Clone creates a deep copy of the static DHCP entry
func (e *StaticDHCPEntry) Clone() *StaticDHCPEntry {
	clone := &StaticDHCPEntry{
		ID:         e.ID,
		Hostname:   e.Hostname,
		Tag:        e.Tag,
		LeaseTime:  e.LeaseTime,
		Comment:    e.Comment,
		Enabled:    e.Enabled,
		LineNumber: e.LineNumber,
		RawLine:    e.RawLine,
	}
	
	// Deep copy MAC address
	if e.MAC != nil {
		clone.MAC = make(net.HardwareAddr, len(e.MAC))
		copy(clone.MAC, e.MAC)
	}
	
	// Deep copy IP address
	if e.IP != nil {
		clone.IP = make(net.IP, len(e.IP))
		copy(clone.IP, e.IP)
	}
	
	return clone
}

// Equal compares two StaticDHCPEntry instances for equality
func (e *StaticDHCPEntry) Equal(other *StaticDHCPEntry) bool {
	if other == nil {
		return false
	}
	
	return e.ID == other.ID &&
		e.GetFormattedMAC() == other.GetFormattedMAC() &&
		e.GetFormattedIP() == other.GetFormattedIP() &&
		e.Hostname == other.Hostname &&
		e.Tag == other.Tag &&
		e.LeaseTime == other.LeaseTime &&
		e.Comment == other.Comment &&
		e.Enabled == other.Enabled
}

// String returns a string representation of the entry
func (e *StaticDHCPEntry) String() string {
	status := "enabled"
	if !e.Enabled {
		status = "disabled"
	}
	
	return fmt.Sprintf("StaticDHCPEntry{MAC: %s, IP: %s, Hostname: %s, Status: %s}",
		e.GetFormattedMAC(),
		e.GetFormattedIP(),
		e.Hostname,
		status)
}

// IsValid performs a quick validation check
func (e *StaticDHCPEntry) IsValid() bool {
	return e.Validate() == nil
}

// GetDisplayName returns a user-friendly display name for the entry
func (e *StaticDHCPEntry) GetDisplayName() string {
	if e.Hostname != "" {
		return e.Hostname
	}
	if e.IP != nil {
		return e.IP.String()
	}
	return e.GetFormattedMAC()
}

// ===== Helper Functions =====

// NormalizeMACAddress converts MAC address to standard format
func NormalizeMACAddress(macStr string) (string, error) {
	if macStr == "" {
		return "", fmt.Errorf("MAC address cannot be empty")
	}
	
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return "", fmt.Errorf("invalid MAC address format: %w", err)
	}
	
	// Return in uppercase with colons format
	return strings.ToUpper(mac.String()), nil
}

// ValidateMACAddress checks if a MAC address string is valid
func ValidateMACAddress(macStr string) error {
	if macStr == "" {
		return fmt.Errorf("MAC address cannot be empty")
	}
	
	_, err := net.ParseMAC(macStr)
	if err != nil {
		return fmt.Errorf("invalid MAC address format: %w", err)
	}
	
	return nil
}

// ValidateIPAddress checks if an IP address string is valid IPv4
func ValidateIPAddress(ipStr string) error {
	if ipStr == "" {
		return nil // IP is optional
	}
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format: %s", ipStr)
	}
	
	// Check if it's IPv4
	if ip.To4() == nil {
		return fmt.Errorf("only IPv4 addresses are supported")
	}
	
	return nil
}

// ===== Example Usage Documentation =====

/*
API Usage Examples:

1. Get all static entries:
   GET /api/static

2. Add a new entry:
   POST /api/static
   {
     "action": "add",
     "entry": {
       "mac": "AA:BB:CC:DD:EE:FF",
       "ip": "192.168.1.100",
       "hostname": "my-device",
       "tag": "trusted",
       "comment": "My important device",
       "enabled": true
     }
   }

3. Update an existing entry:
   POST /api/static
   {
     "action": "update",
     "id": "entry_1234567890",
     "entry": {
       "mac": "AA:BB:CC:DD:EE:FF",
       "ip": "192.168.1.101",
       "hostname": "my-device-updated",
       "enabled": true
     }
   }

4. Delete an entry:
   POST /api/static
   {
     "action": "delete",
     "id": "entry_1234567890"
   }

5. Enable/disable an entry:
   POST /api/static
   {
     "action": "enable",
     "id": "entry_1234567890"
   }

6. Save configuration to file:
   POST /api/static
   {
     "action": "save"
   }

7. Reload configuration from file:
   POST /api/static
   {
     "action": "reload"
   }

8. Validate configuration:
   POST /api/static
   {
     "action": "validate"
   }

9. Filter entries:
   POST /api/static
   {
     "action": "list",
     "filter": {
       "enabled": "true",
       "mac": "AA:BB",
       "hostname": "device"
     }
   }

MAC Address Format:
- Input: Accepts AA:BB:CC:DD:EE:FF, AA-BB-CC-DD-EE-FF, or AABBCCDDEEFF
- Output: Always normalized to AA:BB:CC:DD:EE:FF (uppercase with colons)
- Storage: Stored as net.HardwareAddr for consistency
- Display: Always shown in standard format
*/