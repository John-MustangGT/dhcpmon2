// ===== internal/mac/database.go =====
package mac

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	
	"dhcpmon/pkg/models"
)

// Database handles MAC address OUI lookups
type Database struct {
	cache  map[string]*models.OUIEntry
	file   *os.File
	mu     sync.RWMutex
	preloaded bool
}

// NewDatabase creates a new MAC database instance
func NewDatabase(filename string, preload bool) (*Database, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open MAC database: %w", err)
	}

	db := &Database{
		cache: make(map[string]*models.OUIEntry),
		file:  file,
	}

	// Initialize default entries
	db.initializeDefaults()

	if preload {
		if err := db.preloadDatabase(); err != nil {
			log.Printf("Warning: failed to preload MAC database: %v", err)
		}
	}

	return db, nil
}

// initializeDefaults sets up default OUI entries for unknown and private MACs
func (db *Database) initializeDefaults() {
	db.cache["UNKNOWN"] = &models.OUIEntry{
		OUI:     "00:00:00:00:00:00",
		Private: false,
		Company: "UNKNOWN",
		Address: "UNKNOWN",
	}

	privateMAC := &models.OUIEntry{
		Private: true,
		Company: "Local/Privacy MAC",
		Address: "UNKNOWN",
	}

	// Private MAC patterns
	patterns := []string{"PRIVATE-2", "PRIVATE-6", "PRIVATE-A", "PRIVATE-E"}
	for _, pattern := range patterns {
		db.cache[pattern] = privateMAC
	}
}

// preloadDatabase loads all MAC entries into memory
func (db *Database) preloadDatabase() error {
	db.file.Seek(0, 0)
	scanner := bufio.NewScanner(db.file)
	
	count := 0
	for scanner.Scan() {
		var entry models.OUIEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		
		prefix := strings.ToUpper(entry.OUI)
		db.cache[prefix] = &entry
		count++
	}
	
	db.preloaded = true
	log.Printf("Preloaded %d MAC entries", count)
	return scanner.Err()
}

// Lookup finds OUI information for a MAC address
func (db *Database) Lookup(mac string) *models.OUIEntry {
	mac = strings.ToUpper(mac)
	
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	// Try cache first with progressively shorter prefixes
	for i := len(mac); i >= 0; i-- {
		if entry, exists := db.cache[mac[0:i]]; exists {
			return entry
		}
	}
	
	// Check for private MAC patterns
	if len(mac) > 1 {
		switch mac[1] {
		case '2':
			return db.cache["PRIVATE-2"]
		case '6':
			return db.cache["PRIVATE-6"]
		case 'A':
			return db.cache["PRIVATE-A"]
		case 'E':
			return db.cache["PRIVATE-E"]
		}
	}
	
	// If not preloaded, search the file
	if !db.preloaded {
		if entry := db.searchFile(mac); entry != nil {
			return entry
		}
	}
	
	return db.cache["UNKNOWN"]
}

// searchFile searches the database file for a MAC prefix
func (db *Database) searchFile(mac string) *models.OUIEntry {
	db.file.Seek(0, 0)
	scanner := bufio.NewScanner(db.file)
	
	for scanner.Scan() {
		var entry models.OUIEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		
		prefix := strings.ToUpper(entry.OUI)
		if strings.HasPrefix(mac, prefix) {
			// Cache the result
			db.mu.Lock()
			db.cache[prefix] = &entry
			db.mu.Unlock()
			return &entry
		}
	}
	
	return nil
}

// Close closes the database file
func (db *Database) Close() error {
	if db.file != nil {
		return db.file.Close()
	}
	return nil
}

