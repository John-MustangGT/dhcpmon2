// ===== internal/monitor/monitor.go =====
package monitor

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	
	"github.com/fsnotify/fsnotify"
	
	config "dhcpmon/internal/config"
	dhcp "dhcpmon/internal/dhcp"
	hosts "dhcpmon/internal/hosts"
	logs "dhcpmon/internal/logs"
	models "dhcpmon/pkg/models"
)

// Monitor handles file monitoring and data management
type Monitor struct {
	cfg        *config.Config
	dhcpParser *dhcp.Parser
	hostsParser *hosts.Parser
	logManager *logs.Manager
	
	dhcpLeases []models.DHCPLease
	hostEntries []models.HostEntry
	
	watcher *fsnotify.Watcher
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// New creates a new monitor instance
func New(cfg *config.Config, dhcpParser *dhcp.Parser) *Monitor {
	return &Monitor{
		cfg:         cfg,
		dhcpParser:  dhcpParser,
		hostsParser: hosts.NewParser(),
		logManager:  logs.NewManager(cfg),
		stopCh:      make(chan struct{}),
	}
}

// Start begins monitoring files
func (m *Monitor) Start() error {
	var err error
	m.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	
	// Initial load
	if err := m.loadDHCPLeases(); err != nil {
		log.Printf("Warning: failed to load DHCP leases: %v", err)
	}
	
	if err := m.loadHostEntries(); err != nil {
		log.Printf("Warning: failed to load host entries: %v", err)
	}
	
	// Start file watching
	go m.watchFiles()
	
	// Add files to watcher
	if err := m.watcher.Add(m.cfg.LeasesFile); err != nil {
		log.Printf("Warning: failed to watch leases file: %v", err)
	}
	
	if err := m.watcher.Add(m.cfg.HostsFile); err != nil {
		log.Printf("Warning: failed to watch hosts file: %v", err)
	}
	
	// Start log manager
	if err := m.logManager.Start(); err != nil {
		log.Printf("Warning: failed to start log manager: %v", err)
	}
	
	return nil
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

// watchFiles monitors file changes
func (m *Monitor) watchFiles() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("File modified: %s", event.Name)
				
				switch event.Name {
				case m.cfg.LeasesFile:
					if err := m.loadDHCPLeases(); err != nil {
						log.Printf("Error reloading DHCP leases: %v", err)
					}
				case m.cfg.HostsFile:
					if err := m.loadHostEntries(); err != nil {
						log.Printf("Error reloading host entries: %v", err)
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

