// ===== internal/logs/manager.go =====
package logs

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"time"
	
	"dhcpmon/internal/config"
	"dhcpmon/pkg/models"
)

const maxLogEntries = 100

// JournalOutput represents systemd journal output
type JournalOutput struct {
	Cursor    string `json:"__CURSOR"`
	Timestamp string `json:"__REALTIME_TIMESTAMP"`
	Message   string `json:"MESSAGE"`
	Transport string `json:"_TRANSPORT"`
}

// Manager handles log collection and storage
type Manager struct {
	cfg    *config.Config
	logs   *list.List
	mu     sync.RWMutex
	stopCh chan struct{}
}

// NewManager creates a new log manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:    cfg,
		logs:   list.New(),
		stopCh: make(chan struct{}),
	}
}

// Start begins log collection
func (m *Manager) Start() error {
	if !m.cfg.SystemD {
		// Start dnsmasq and collect its logs
		go m.startDNSMasq()
	}
	return nil
}

// Stop stops log collection
func (m *Manager) Stop() {
	close(m.stopCh)
}

// GetLogs returns current log entries
func (m *Manager) GetLogs() []models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var entries []models.LogEntry
	for e := m.logs.Front(); e != nil; e = e.Next() {
		entries = append(entries, *(e.Value.(*models.LogEntry)))
	}
	
	return entries
}

// GetSystemdLogs retrieves logs from systemd journal
func (m *Manager) GetSystemdLogs() ([]models.LogEntry, error) {
	cmd := exec.Command("/bin/journalctl",
		"--unit=dnsmasq.service",
		"--output=json")
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get systemd logs: %w", err)
	}
	
	var entries []models.LogEntry
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		
		var journalEntry JournalOutput
		if err := json.Unmarshal([]byte(line), &journalEntry); err != nil {
			continue
		}
		
		timestamp, err := strconv.ParseInt(journalEntry.Timestamp, 10, 64)
		if err != nil {
			continue
		}
		
		entry := models.LogEntry{
			Timestamp: time.Unix(timestamp/1000000, timestamp%1000000),
			UnixTime:  timestamp / 1000,
			Channel:   journalEntry.Transport,
			Message:   journalEntry.Message,
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// addLogEntry adds a new log entry to the collection
func (m *Manager) addLogEntry(entry *models.LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Remove old entries if we exceed the limit
	if m.logs.Len() >= maxLogEntries {
		m.logs.Remove(m.logs.Front())
	}
	
	m.logs.PushBack(entry)
}

// startDNSMasq starts dnsmasq and collects its output
func (m *Manager) startDNSMasq() {
	cmdArgs := []string{
		m.cfg.DNSMasq,
		"--keep-in-foreground",
		"--conf-dir=/etc/dnsmasq.d,*conf",
	}
	
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %v", err)
		return
	}
	
	log.Printf("Starting dnsmasq: %v", cmdArgs)
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start dnsmasq: %v", err)
		return
	}
	
	// Start log scanners
	go m.scanLogs(stdout, "stdout")
	go m.scanLogs(stderr, "stderr")
	
	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		log.Printf("dnsmasq exited with error: %v", err)
	}
}

// scanLogs scans output from a reader and creates log entries
func (m *Manager) scanLogs(reader io.Reader, channel string) {
	scanner := bufio.NewScanner(reader)
	
	for scanner.Scan() {
		entry := &models.LogEntry{
			Timestamp: time.Now(),
			UnixTime:  time.Now().Unix(),
			Channel:   channel,
			Message:   scanner.Text(),
		}
		
		m.addLogEntry(entry)
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning %s: %v", channel, err)
	}
}

