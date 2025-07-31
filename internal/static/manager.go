// ===== internal/static/manager.go =====
package static

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	
	"dhcpmon/pkg/models"
)

// Manager handles static DHCP configuration management
type Manager struct {
	parser     *Parser
	filename   string
	entries    []models.StaticDHCPEntry
	mu         sync.RWMutex
	lastModify time.Time
}

// NewManager creates a new static DHCP manager
func NewManager(filename string) *Manager {
	return &Manager{
		parser:   NewParser(),
		filename: filename,
		entries:  make([]models.StaticDHCPEntry, 0),
	}
}

// Load loads static DHCP entries from the configuration file
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	entries, err := m.parser.ParseFile(m.filename)
	if err != nil {
		return fmt.Errorf("failed to load static entries: %w", err)
	}
	
	m.entries = entries
	m.lastModify = time.Now()
	
	log.Printf("Loaded %d static DHCP entries from %s", len(entries), m.filename)
	return nil
}

// Save saves static DHCP entries to the configuration file
func (m *Manager) Save() error {
	m.mu.RLock()
	entries := make([]models.StaticDHCPEntry, len(m.entries))
	copy(entries, m.entries)
	m.mu.RUnlock()
	
	if err := m.parser.WriteFile(m.filename, entries); err != nil {
		return fmt.Errorf("failed to save static entries: %w", err)
	}
	
	m.mu.Lock()
	m.lastModify = time.Now()
	m.mu.Unlock()
	
	log.Printf("Saved %d static DHCP entries to %s", len(entries), m.filename)
	return nil
}

// GetAll returns all static DHCP entries
func (m *Manager) GetAll() []models.StaticDHCPEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	entries := make([]models.StaticDHCPEntry, len(m.entries))
	copy(entries, m.entries)
	return entries
}

// GetByID returns a static DHCP entry by ID
func (m *Manager) GetByID(id string) (*models.StaticDHCPEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, entry := range m.entries {
		if entry.ID == id {
			entryCopy := entry
			return &entryCopy, nil
		}
	}
	
	return nil, fmt.Errorf("entry with ID %s not found", id)
}

// Add adds a new static DHCP entry
func (m *Manager) Add(entry models.StaticDHCPEntry) error {
	if err := entry.Validate(); err != nil {
		return fmt.Errorf("invalid entry: %w", err)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check for duplicate MAC addresses
	for _, existing := range m.entries {
		if existing.MAC.String() == entry.MAC.String() && existing.Enabled {
			return fmt.Errorf("MAC address %s already exists", entry.MAC.String())
		}
	}
	
	// Check for duplicate IP addresses
	if entry.IP != nil {
		for _, existing := range m.entries {
			if existing.IP != nil && existing.IP.Equal(entry.IP) && existing.Enabled {
				return fmt.Errorf("IP address %s already exists", entry.IP.String())
			}
		}
	}
	
	// Generate new ID
	entry.ID = fmt.Sprintf("entry_%d", time.Now().Unix())
	entry.LineNumber = len(m.entries) + 1
	
	m.entries = append(m.entries, entry)
	return nil
}

// Update updates an existing static DHCP entry
func (m *Manager) Update(id string, updatedEntry models.StaticDHCPEntry) error {
	if err := updatedEntry.Validate(); err != nil {
		return fmt.Errorf("invalid entry: %w", err)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, entry := range m.entries {
		if entry.ID == id {
			// Check for duplicate MAC (excluding current entry)
			for j, existing := range m.entries {
				if j != i && existing.MAC.String() == updatedEntry.MAC.String() && existing.Enabled {
					return fmt.Errorf("MAC address %s already exists", updatedEntry.MAC.String())
				}
			}
			
			// Check for duplicate IP (excluding current entry)
			if updatedEntry.IP != nil {
				for j, existing := range m.entries {
					if j != i && existing.IP != nil && existing.IP.Equal(updatedEntry.IP) && existing.Enabled {
						return fmt.Errorf("IP address %s already exists", updatedEntry.IP.String())
					}
				}
			}
			
			// Preserve original fields
			updatedEntry.ID = entry.ID
			updatedEntry.LineNumber = entry.LineNumber
			
			m.entries[i] = updatedEntry
			return nil
		}
	}
	
	return fmt.Errorf("entry with ID %s not found", id)
}

// Delete deletes a static DHCP entry
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, entry := range m.entries {
		if entry.ID == id {
			// Remove entry from slice
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	
	return fmt.Errorf("entry with ID %s not found", id)
}

// Enable enables a static DHCP entry
func (m *Manager) Enable(id string) error {
	return m.setEnabled(id, true)
}

// Disable disables a static DHCP entry
func (m *Manager) Disable(id string) error {
	return m.setEnabled(id, false)
}

// setEnabled sets the enabled state of an entry
func (m *Manager) setEnabled(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, entry := range m.entries {
		if entry.ID == id {
			m.entries[i].Enabled = enabled
			return nil
		}
	}
	
	return fmt.Errorf("entry with ID %s not found", id)
}

// GetByMAC returns entries with a specific MAC address
func (m *Manager) GetByMAC(mac net.HardwareAddr) []models.StaticDHCPEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var results []models.StaticDHCPEntry
	for _, entry := range m.entries {
		if entry.MAC.String() == mac.String() {
			results = append(results, entry)
		}
	}
	
	return results
}

// GetByIP returns entries with a specific IP address
func (m *Manager) GetByIP(ip net.IP) []models.StaticDHCPEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var results []models.StaticDHCPEntry
	for _, entry := range m.entries {
		if entry.IP != nil && entry.IP.Equal(ip) {
			results = append(results, entry)
		}
	}
	
	return results
}

// Validate validates all entries and returns any errors
func (m *Manager) Validate() []error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var errors []error
	macMap := make(map[string]int)
	ipMap := make(map[string]int)
	
	for i, entry := range m.entries {
		// Validate individual entry
		if err := entry.Validate(); err != nil {
			errors = append(errors, fmt.Errorf("entry %d: %w", i+1, err))
		}
		
		// Check for duplicate MACs among enabled entries
		if entry.Enabled && entry.MAC != nil {
			macStr := entry.MAC.String()
			if existing, exists := macMap[macStr]; exists {
				errors = append(errors, fmt.Errorf("duplicate MAC %s in entries %d and %d", macStr, existing+1, i+1))
			} else {
				macMap[macStr] = i
			}
		}
		
		// Check for duplicate IPs among enabled entries
		if entry.Enabled && entry.IP != nil {
			ipStr := entry.IP.String()
			if existing, exists := ipMap[ipStr]; exists {
				errors = append(errors, fmt.Errorf("duplicate IP %s in entries %d and %d", ipStr, existing+1, i+1))
			} else {
				ipMap[ipStr] = i
			}
		}
	}
	
	return errors
}

