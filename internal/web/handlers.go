// ===== internal/web/handlers.go (Fixed) =====
package web

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"
)

// DHCPLeaseJSON represents a DHCP lease in JSON format
type DHCPLeaseJSON struct {
	Expire string        `json:"expire"`
	Remain string        `json:"remain"`
	Delta  time.Duration `json:"delta"`
	MAC    string        `json:"mac"`
	Info   interface{}   `json:"info"`
	IP     string        `json:"ip"`
	IPSort uint32        `json:"ipSort"`
	Name   string        `json:"name"`
	ID     string        `json:"id"`
	Tag    string        `json:"tag"`
	Static bool          `json:"static"`
}

// LogEntryJSON represents a log entry in JSON format
type LogEntryJSON struct {
	Timestamp string `json:"when"`
	UnixTime  int64  `json:"utime"`
	Channel   string `json:"channel"`
	Message   string `json:"message"`
}

// handleLeasesAPI handles DHCP leases API requests
func (s *Server) handleLeasesAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	log.Printf("Handling leases API request")
	
	leases := s.monitor.GetDHCPLeases()
	log.Printf("Found %d DHCP leases", len(leases))
	
	jsonLeases := make([]DHCPLeaseJSON, len(leases))
	
	for i, lease := range leases {
		remainStr := "Infinite"
		expireStr := "Never"
		
		if !lease.Static {
			remainStr = lease.Remain.Round(time.Second).String()
			expireStr = lease.Expire.Format("2006-01-02 15:04:05")
		}
		
		// Handle nil IP addresses
		ipStr := ""
		var ipSort uint32 = 0
		if lease.IP != nil {
			ipStr = lease.IP.String()
			ipSort = s.ipToInt(lease.IP)
		}
		
		// Handle nil MAC addresses
		macStr := ""
		if lease.MAC != nil {
			macStr = lease.MAC.String()
		}
		
		jsonLeases[i] = DHCPLeaseJSON{
			Expire: expireStr,
			Remain: remainStr,
			Delta:  lease.Remain,
			MAC:    macStr,
			Info:   lease.Info,
			IP:     ipStr,
			IPSort: ipSort,
			Name:   lease.Name,
			ID:     lease.ID,
			Tag:    lease.Tag,
			Static: lease.Static,
		}
	}
	
	response := map[string]interface{}{"data": jsonLeases}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode leases JSON: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
	}
}

// handleHostsAPI handles hosts file API requests
func (s *Server) handleHostsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	log.Printf("Handling hosts API request")
	
	hosts := s.monitor.GetHostEntries()
	log.Printf("Found %d host entries", len(hosts))
	
	response := map[string]interface{}{"data": hosts}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode hosts JSON: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
	}
}

// handleLogsAPI handles logs API requests
func (s *Server) handleLogsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	log.Printf("Handling logs API request")
	
	var logs interface{}
	
	if s.cfg.SystemD {
		// Get logs from systemd journal
		logEntries, sysErr := s.monitor.GetSystemdLogs()
		if sysErr != nil {
			log.Printf("Failed to get systemd logs: %v", sysErr)
			logs = []LogEntryJSON{}
		} else {
			log.Printf("Found %d systemd log entries", len(logEntries))
			jsonLogs := make([]LogEntryJSON, len(logEntries))
			for i, entry := range logEntries {
				jsonLogs[i] = LogEntryJSON{
					Timestamp: entry.Timestamp.Format(time.RFC3339),
					UnixTime:  entry.UnixTime,
					Channel:   entry.Channel,
					Message:   entry.Message,
				}
			}
			logs = jsonLogs
		}
	} else {
		// Get logs from local collection
		logEntries := s.monitor.GetLogs()
		log.Printf("Found %d local log entries", len(logEntries))
		jsonLogs := make([]LogEntryJSON, len(logEntries))
		for i, entry := range logEntries {
			jsonLogs[i] = LogEntryJSON{
				Timestamp: entry.Timestamp.Format(time.RFC3339),
				UnixTime:  entry.UnixTime,
				Channel:   entry.Channel,
				Message:   entry.Message,
			}
		}
		logs = jsonLogs
	}
	
	response := map[string]interface{}{"data": logs}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode logs JSON: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
	}
}

// handleRemoveAPI handles remove requests for DHCP entries
func (s *Server) handleRemoveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	
	dataStr := r.FormValue("data")
	if dataStr == "" {
		http.Error(w, `{"error":"Missing data parameter"}`, http.StatusBadRequest)
		return
	}
	
	log.Printf("Remove request received: %s", dataStr)
	
	// Parse the JSON data to understand what to remove
	var removeData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &removeData); err != nil {
		log.Printf("Failed to parse remove data: %v", err)
		http.Error(w, `{"error":"Invalid JSON data"}`, http.StatusBadRequest)
		return
	}
	
	// TODO: Implement actual removal logic here
	// This would involve modifying the static hosts file or 
	// sending commands to dnsmasq to remove dynamic leases
	
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "success",
		"message": "Remove request processed",
	}
	json.NewEncoder(w).Encode(response)
}

// handleEditAPI handles edit requests for DHCP entries
func (s *Server) handleEditAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	
	dataStr := r.FormValue("data")
	if dataStr == "" {
		http.Error(w, `{"error":"Missing data parameter"}`, http.StatusBadRequest)
		return
	}
	
	log.Printf("Edit request received: %s", dataStr)
	
	// Parse the JSON data
	var editData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &editData); err != nil {
		log.Printf("Failed to parse edit data: %v", err)
		http.Error(w, `{"error":"Invalid JSON data"}`, http.StatusBadRequest)
		return
	}
	
	// TODO: Implement actual edit logic here
	// This would involve updating configuration files
	
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "success",
		"message": "Edit request processed",
	}
	json.NewEncoder(w).Encode(response)
}

// ipToInt converts IP to integer with better error handling
func (s *Server) ipToInt(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	
	// Handle both IPv4 and IPv6-mapped IPv4
	if len(ip) == 16 {
		ip = ip[12:16]
	}
	
	if len(ip) != 4 {
		return 0
	}
	
	return uint32(ip[0])<<24 + uint32(ip[1])<<16 + uint32(ip[2])<<8 + uint32(ip[3])
}
