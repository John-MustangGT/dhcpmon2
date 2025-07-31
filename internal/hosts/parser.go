// ===== internal/hosts/parser.go =====
package hosts

import (
	"strings"
	"unicode"
	
	"dhcpmon/pkg/models"
)

// Parser handles hosts file parsing
type Parser struct{}

// NewParser creates a new hosts parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseHosts parses hosts file content
func (p *Parser) ParseHosts(content string) ([]models.HostEntry, error) {
	var entries []models.HostEntry
	
	for _, line := range strings.Split(content, "\n") {
		line = p.stripComment(strings.TrimSpace(line))
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		
		entry := models.HostEntry{
			IP:   fields[0],
			Name: fields[1],
		}
		
		if len(fields) > 2 {
			entry.Alias = fields[2:]
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// stripComment removes comments from a line
func (p *Parser) stripComment(line string) string {
	if idx := strings.IndexAny(line, "#;"); idx >= 0 {
		return strings.TrimRightFunc(line[:idx], unicode.IsSpace)
	}
	return line
}

