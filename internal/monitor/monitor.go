// ===== internal/monitor/monitor.go (Updated) =====
package monitor

import (
	"fmt"
	"path/filepath"
	"log"
	"os"
	"net"
	"sync"
	"time"
	
	"github.com/fsnotify/fsnotify"
	
	"dhcpmon/internal/config"
	"dhcpmon/internal/dhcp"
	"dhcpmon/internal/hosts"
	"dhcpmon/internal/logs"
	"dhcpmon/internal/static"
	"dhcpmon/pkg/models"
)

// Monitor handles file monitoring and data management
type Monitor struct {
	cfg        *config.Config
	dhcpParser *dhcp.Parser
	hostsParser *hosts.Parser
	logManager *logs.Manager
	staticManager *static.Manager
	
	dhcpLeases []models.DHCPLease
	hostEntries []models.HostEntry
	
	watcher *fsnotify.Watcher
	mu      sync.RWMutex
	staticMu sync.RWMutex
	stopCh  chan struct{}
}

// New creates a new monitor instance
func New(cfg *config.Config, dhcpParser *dhcp.Parser) *Monitor {
	return &Monitor{
		cfg:         cfg,
		dhcpParser:  dhcpParser,
		hostsParser: hosts.NewParser(),
		logManager:  logs.NewManager(cfg),
		staticManager: static.NewManager(cfg.StaticFile),
		stopCh:      make(chan struct{}),
	}
}

// Start begins monitoring files
// Fixed version of Monitor.Start() method
// Replace in internal/monitor/monitor.go

func (m *Monitor) Start() error {
	var err error
	m.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Initial load (with better error handling)
	if err := m.loadDHCPLeases(); err != nil {
		log.Printf("Warning: failed to load DHCP leases: %v", err)
	}

	if err := m.loadHostEntries(); err != nil {
		log.Printf("Warning: failed to load host entries: %v", err)
	}

	// Load static entries
	if err := m.staticManager.Load(); err != nil {
		log.Printf("Warning: failed to load static entries: %v", err)
	}

	// Start file watching goroutine BEFORE adding files
	go m.watchFiles()

	// Add files to watcher with existence checks
	m.addFileToWatcher(m.cfg.LeasesFile, "leases file")
	m.addFileToWatcher(m.cfg.HostsFile, "hosts file")

	if m.cfg.StaticFile != "" {
		m.addFileToWatcher(m.cfg.StaticFile, "static file")
	}

	// Start log manager
	if err := m.logManager.Start(); err != nil {
		log.Printf("Warning: failed to start log manager: %v", err)
	}

	return nil
}

// Helper method to safely add files to watcher
func (m *Monitor) addFileToWatcher(filePath, description string) {

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Warning: %s does not exist: %s", description, filePath)

		// Try to create parent directory and empty file
		if err := m.ensureFileExists(filePath); err != nil {
			log.Printf("Warning: could not create %s: %v", description, err)
			return
		}
	}

	// Add to watcher
	if err := m.watcher.Add(filePath); err != nil {
		log.Printf("Warning: failed to watch %s (%s): %v", description, filePath, err)
	} else {
	}
}

// Helper to ensure file exists for watching
func (m *Monitor) ensureFileExists(filePath string) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create empty file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		file.Close()
		log.Printf("Created empty file: %s", filePath)
	}

	return nil
}

// Fixed watchFiles method with better error handling
func (m *Monitor) watchFiles() {

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("File modified: %s", event.Name)

				// Use absolute paths for comparison
				absEventPath, _ := filepath.Abs(event.Name)
				absLeasesPath, _ := filepath.Abs(m.cfg.LeasesFile)
				absHostsPath, _ := filepath.Abs(m.cfg.HostsFile)
				absStaticPath, _ := filepath.Abs(m.cfg.StaticFile)

				switch absEventPath {
				case absLeasesPath:
					if err := m.loadDHCPLeases(); err != nil {
						log.Printf("Error reloading DHCP leases: %v", err)
					}
				case absHostsPath:
					if err := m.loadHostEntries(); err != nil {
						log.Printf("Error reloading host entries: %v", err)
					}
				case absStaticPath:
					if err := m.staticManager.Load(); err != nil {
						log.Printf("Error reloading static entries: %v", err)
					}
				}
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)

		case <-m.stopCh:
			return
		}
	}
}

// Stop stops monitoring
func (m *Monitor) Stop() {
	close(m.stopCh)
	if m.watcher != nil {
		m.watcher.Close()
	}
	if m.logManager != nil {
		m.logManager.Stop()
	}
}

// GetDHCPLeases returns current DHCP leases
func (m *Monitor) GetDHCPLeases() []models.DHCPLease {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Update remaining time
	leases := make([]models.DHCPLease, len(m.dhcpLeases))
	for i, lease := range m.dhcpLeases {
		leases[i] = lease
		if !lease.Static {
			leases[i].Remain = time.Until(lease.Expire).Truncate(time.Second)
		}
	}
	
	return leases
}

// GetHostEntries returns current host entries
func (m *Monitor) GetHostEntries() []models.HostEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	entries := make([]models.HostEntry, len(m.hostEntries))
	copy(entries, m.hostEntries)
	return entries
}

// GetLogs returns current logs
func (m *Monitor) GetLogs() []models.LogEntry {
	return m.logManager.GetLogs()
}

// ===== Static DHCP Management Methods =====

// GetStaticEntries returns all static DHCP entries
func (m *Monitor) GetStaticEntries() []models.StaticDHCPEntry {
	return m.staticManager.GetAll()
}

// GetStaticEntryByID returns a static DHCP entry by ID
func (m *Monitor) GetStaticEntryByID(id string) (*models.StaticDHCPEntry, error) {
	return m.staticManager.GetByID(id)
}

// AddStaticEntry adds a new static DHCP entry
func (m *Monitor) AddStaticEntry(entry models.StaticDHCPEntry) error {
	return m.staticManager.Add(entry)
}

// UpdateStaticEntry updates an existing static DHCP entry
func (m *Monitor) UpdateStaticEntry(id string, entry models.StaticDHCPEntry) error {
	return m.staticManager.Update(id, entry)
}

// DeleteStaticEntry deletes a static DHCP entry
func (m *Monitor) DeleteStaticEntry(id string) error {
	return m.staticManager.Delete(id)
}

// EnableStaticEntry enables a static DHCP entry
func (m *Monitor) EnableStaticEntry(id string) error {
	return m.staticManager.Enable(id)
}

// DisableStaticEntry disables a static DHCP entry
func (m *Monitor) DisableStaticEntry(id string) error {
	return m.staticManager.Disable(id)
}

// SaveStaticEntries saves static DHCP entries to file
func (m *Monitor) SaveStaticEntries() error {
	return m.staticManager.Save()
}

// ReloadStaticEntries reloads static DHCP entries from file
func (m *Monitor) ReloadStaticEntries() error {
	return m.staticManager.Load()
}

// ValidateStaticEntries validates all static DHCP entries
func (m *Monitor) ValidateStaticEntries() []error {
	return m.staticManager.Validate()
}

// GetStaticEntriesByMAC returns static entries for a MAC address
func (m *Monitor) GetStaticEntriesByMAC(mac string) ([]models.StaticDHCPEntry, error) {
	parsedMAC, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address: %w", err)
	}
	
	return m.staticManager.GetByMAC(parsedMAC), nil
}

// GetStaticEntriesByIP returns static entries for an IP address
func (m *Monitor) GetStaticEntriesByIP(ip string) ([]models.StaticDHCPEntry, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}
	
	return m.staticManager.GetByIP(parsedIP), nil
}

// loadDHCPLeases loads DHCP leases from file
func (m *Monitor) loadDHCPLeases() error {
	content, err := os.ReadFile(m.cfg.LeasesFile)
	if err != nil {
		return fmt.Errorf("failed to read leases file: %w", err)
	}
	
	leases, err := m.dhcpParser.ParseLeases(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse leases: %w", err)
	}
	

	m.mu.Lock()
	m.dhcpLeases = leases
	m.mu.Unlock()
	
	log.Printf("Loaded %d DHCP leases", len(leases))
	return nil
}

// loadHostEntries loads host entries from file
func (m *Monitor) loadHostEntries() error {
	content, err := os.ReadFile(m.cfg.HostsFile)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}
	
	entries, err := m.hostsParser.ParseHosts(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse hosts: %w", err)
	}
	
	m.mu.Lock()
	m.hostEntries = entries
	m.mu.Unlock()
	
	log.Printf("Loaded %d host entries", len(entries))
	return nil
}


// GetSystemdLogs returns logs from systemd journal
func (m *Monitor) GetSystemdLogs() ([]models.LogEntry, error) {
	return m.logManager.GetSystemdLogs()
}
