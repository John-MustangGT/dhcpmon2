// ===== internal/dhcp/parser.go =====
package dhcp

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	
	"dhcpmon/pkg/models"
	"dhcpmon/internal/mac"
)

// Parser handles DHCP lease file parsing
type Parser struct {
	macDB      *mac.Database
	staticFile string
}

// NewParser creates a new DHCP parser
func NewParser(macDB *mac.Database, staticFile string) *Parser {
	return &Parser{
		macDB:      macDB,
		staticFile: staticFile,
	}
}

// ParseLeases parses DHCP lease data from string content
func (p *Parser) ParseLeases(content string) ([]models.DHCPLease, error) {
	var leases []models.DHCPLease
	
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		
		lease, err := p.parseLeaseLine(fields)
		if err != nil {
			continue // Skip invalid lines
		}
		
		leases = append(leases, lease)
	}
	
	// Add static leases if configured
	if p.staticFile != "" {
		staticLeases, err := p.parseStaticLeases()
		if err == nil {
			leases = append(leases, staticLeases...)
		}
	}
	
	return leases, nil
}

// parseLeaseLine parses a single lease line
func (p *Parser) parseLeaseLine(fields []string) (models.DHCPLease, error) {
	var lease models.DHCPLease
	
	// Parse expire time
	if timestamp, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
		lease.Expire = time.Unix(timestamp, 0)
		lease.Remain = time.Until(lease.Expire)
	}
	
	// Parse MAC address
	if mac, err := net.ParseMAC(fields[1]); err == nil {
		lease.MAC = mac
		lease.Info = p.macDB.Lookup(fields[1])
	}
	
	// Parse IP address
	lease.IP = net.ParseIP(fields[2])
	
	// Parse name and ID
	lease.Name = fields[3]
	if len(fields) > 4 {
		lease.ID = fields[4]
	}
	
	return lease, nil
}

// parseStaticLeases parses static DHCP reservations
func (p *Parser) parseStaticLeases() ([]models.DHCPLease, error) {
	file, err := os.Open(p.staticFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var leases []models.DHCPLease
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		lease, err := p.parseStaticLine(line)
		if err != nil {
			continue
		}
		
		leases = append(leases, lease)
	}
	
	return leases, scanner.Err()
}

// parseStaticLine parses a single static DHCP line
func (p *Parser) parseStaticLine(line string) (models.DHCPLease, error) {
	var lease models.DHCPLease
	
	// Remove comments
	if idx := strings.IndexAny(line, "#;"); idx >= 0 {
		line = strings.TrimRightFunc(line[:idx], unicode.IsSpace)
	}
	
	parts := strings.Split(line, "=")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "dhcp-host" {
		return lease, fmt.Errorf("not a dhcp-host line")
	}
	
	values := strings.Split(parts[1], ",")
	if len(values) < 2 {
		return lease, fmt.Errorf("insufficient values")
	}
	
	// Parse MAC address (first field)
	mac, err := net.ParseMAC(values[0])
	if err != nil {
		return lease, fmt.Errorf("invalid MAC address: %w", err)
	}
	
	lease.MAC = mac
	lease.Info = p.macDB.Lookup(values[0])
	lease.Static = true
	
	// Parse remaining fields
	for i := 1; i < len(values); i++ {
		value := values[i]
		
		// Check for tag
		if strings.Contains(value, ":") {
			tagParts := strings.Split(strings.ToLower(value), ":")
			if len(tagParts) == 2 && (tagParts[0] == "set" || tagParts[0] == "tag") {
				lease.Tag = tagParts[1]
				continue
			}
		}
		
		// Check for IP address
		if ip := net.ParseIP(value); ip != nil {
			lease.IP = ip
			continue
		}
		
		// Otherwise, it's the hostname
		lease.Name = value
		lease.ID = value
	}
	
	// Set infinite lease time for static entries
	lease.Expire = time.Now().Add(time.Hour * 24 * 365 * 10) // 10 years
	lease.Remain = time.Hour * 24 * 365 * 10
	
	return lease, nil
}

