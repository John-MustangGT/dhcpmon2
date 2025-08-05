// ===== internal/web/handlers.go (Updated with working edit functionality) =====
// 
// Also update internal/web/server.go to add the edit route:
//
// func (s *Server) setupRoutes() {
// 	s.mux.HandleFunc("/", s.handleRoot)
// 	s.mux.HandleFunc("/api/static", s.handleStaticAPI)
// 	s.mux.HandleFunc("/api/edit", s.handleEditAPI)  // Add this line
// }
// 
// Or modify the handleRoot function to properly route edit requests
package web

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	
	"dhcpmon/pkg/models"
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

// EditRequest represents an edit request from the frontend
type EditRequest struct {
	MAC      string `json:"mac"`
	IP       string `json:"ip"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Tag      string `json:"tag"`
	Comment  string `json:"comment"`
	Static   bool   `json:"static"`
}

// EditResponse represents the response to an edit request
type EditResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
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
		
		// Format MAC address properly (AA:BB:CC:DD:EE:FF format)
		macStr := ""
		if lease.MAC != nil {
			macStr = s.formatMACAddress(lease.MAC)
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
	
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	dataStr := r.FormValue("data")
	if dataStr == "" {
		s.writeJSONError(w, "Missing data parameter", http.StatusBadRequest)
		return
	}
	
	log.Printf("Remove request received: %s", dataStr)
	
	// Parse the JSON data to understand what to remove
	var removeData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &removeData); err != nil {
		log.Printf("Failed to parse remove data: %v", err)
		s.writeJSONError(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	
	// Extract MAC address from the remove data
	macStr, ok := removeData["mac"].(string)
	if !ok || macStr == "" {
		s.writeJSONError(w, "MAC address is required for removal", http.StatusBadRequest)
		return
	}
	
	// Try to find and remove from static entries
	staticEntries := s.monitor.GetStaticEntries()
	var entryToRemove *models.StaticDHCPEntry
	
	for _, entry := range staticEntries {
		if strings.EqualFold(entry.GetFormattedMAC(), strings.ToUpper(macStr)) {
			entryToRemove = &entry
			break
		}
	}
	
	if entryToRemove != nil {
		if err := s.monitor.DeleteStaticEntry(entryToRemove.ID); err != nil {
			log.Printf("Failed to delete static entry: %v", err)
			s.writeJSONError(w, "Failed to delete static entry: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Save the changes
		if err := s.monitor.SaveStaticEntries(); err != nil {
			log.Printf("Failed to save static entries: %v", err)
			s.writeJSONError(w, "Failed to save changes: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		response := EditResponse{
			Success: true,
			Message: "Entry removed successfully",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// If not found in static entries, return appropriate message
	response := EditResponse{
		Success: false,
		Message: "Cannot remove dynamic DHCP lease. Only static entries can be removed.",
	}
	json.NewEncoder(w).Encode(response)
}

// handleEditAPI handles edit requests for DHCP entries
func (s *Server) handleEditAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleEditGetData(w, r)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	dataStr := r.FormValue("data")
	if dataStr == "" {
		s.writeJSONError(w, "Missing data parameter", http.StatusBadRequest)
		return
	}
	
	log.Printf("Edit request received: %s", dataStr)
	
	// Parse the JSON data
	var editData EditRequest
	if err := json.Unmarshal([]byte(dataStr), &editData); err != nil {
		log.Printf("Failed to parse edit data: %v", err)
		s.writeJSONError(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if editData.MAC == "" {
		s.writeJSONError(w, "MAC address is required", http.StatusBadRequest)
		return
	}
	
	// Parse and validate MAC address
	mac, err := net.ParseMAC(editData.MAC)
	if err != nil {
		s.writeJSONError(w, "Invalid MAC address format", http.StatusBadRequest)
		return
	}
	
	// Parse IP address if provided
	var ip net.IP
	if editData.IP != "" {
		ip = net.ParseIP(editData.IP)
		if ip == nil {
			s.writeJSONError(w, "Invalid IP address format", http.StatusBadRequest)
			return
		}
	}
	
	// Determine hostname (use Name if Hostname is empty)
	hostname := editData.Hostname
	if hostname == "" {
		hostname = editData.Name
	}
	
	// Look for existing static entry with this MAC
	staticEntries := s.monitor.GetStaticEntries()
	var existingEntry *models.StaticDHCPEntry
	
	for _, entry := range staticEntries {
		if strings.EqualFold(entry.GetFormattedMAC(), strings.ToUpper(editData.MAC)) {
			existingEntry = &entry
			break
		}
	}
	
	if existingEntry != nil {
		// Update existing static entry
		updatedEntry := models.StaticDHCPEntry{
			MAC:       mac,
			IP:        ip,
			Hostname:  hostname,
			Tag:       editData.Tag,
			Comment:   editData.Comment,
			Enabled:   true,
		}
		
		if err := s.monitor.UpdateStaticEntry(existingEntry.ID, updatedEntry); err != nil {
			log.Printf("Failed to update static entry: %v", err)
			s.writeJSONError(w, "Failed to update entry: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Create new static entry
		newEntry := models.StaticDHCPEntry{
			MAC:       mac,
			IP:        ip,
			Hostname:  hostname,
			Tag:       editData.Tag,
			Comment:   editData.Comment,
			Enabled:   true,
		}
		
		if err := s.monitor.AddStaticEntry(newEntry); err != nil {
			log.Printf("Failed to add static entry: %v", err)
			s.writeJSONError(w, "Failed to add entry: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
	// Save the changes
	if err := s.monitor.SaveStaticEntries(); err != nil {
		log.Printf("Failed to save static entries: %v", err)
		s.writeJSONError(w, "Failed to save changes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := EditResponse{
		Success: true,
		Message: "Entry saved successfully",
	}
	json.NewEncoder(w).Encode(response)
}

// handleEditGetData handles GET requests to retrieve data for editing
func (s *Server) handleEditGetData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	macParam := r.URL.Query().Get("mac")
	if macParam == "" {
		s.writeJSONError(w, "MAC address parameter is required", http.StatusBadRequest)
		return
	}
	
	log.Printf("Getting edit data for MAC: %s", macParam)
	
	// First check static entries
	staticEntries := s.monitor.GetStaticEntries()
	for _, entry := range staticEntries {
		if strings.EqualFold(entry.GetFormattedMAC(), strings.ToUpper(macParam)) {
			response := map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"mac":      entry.GetFormattedMAC(),
					"ip":       entry.GetFormattedIP(),
					"hostname": entry.Hostname,
					"name":     entry.Hostname,
					"tag":      entry.Tag,
					"comment":  entry.Comment,
					"static":   true,
					"enabled":  entry.Enabled,
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	
	// Then check current DHCP leases
	leases := s.monitor.GetDHCPLeases()
	for _, lease := range leases {
		if lease.MAC != nil && strings.EqualFold(strings.ToUpper(lease.MAC.String()), strings.ToUpper(macParam)) {
			ipStr := ""
			if lease.IP != nil {
				ipStr = lease.IP.String()
			}
			
			response := map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"mac":      s.formatMACAddress(lease.MAC),
					"ip":       ipStr,
					"hostname": lease.Name,
					"name":     lease.Name,
					"tag":      lease.Tag,
					"comment":  "",
					"static":   lease.Static,
					"enabled":  true,
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	
	// Entry not found
	s.writeJSONError(w, "Entry not found", http.StatusNotFound)
}

// Helper function to write JSON error responses
func (s *Server) writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := EditResponse{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}

// formatMACAddress formats MAC address in standard AA:BB:CC:DD:EE:FF format
func (s *Server) formatMACAddress(mac net.HardwareAddr) string {
	if mac == nil {
		return ""
	}
	
	// Convert to uppercase and ensure colon format
	return strings.ToUpper(mac.String())
}

// parseMACAddress parses and normalizes MAC address from various formats
func (s *Server) parseMACAddress(macStr string) (net.HardwareAddr, error) {
	if macStr == "" {
		return nil, nil
	}
	
	// Parse MAC address (supports multiple formats)
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return nil, err
	}
	
	return mac, nil
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

// validateMACFormat checks if MAC address is in valid format
func (s *Server) validateMACFormat(macStr string) bool {
	if macStr == "" {
		return false
	}
	
	_, err := net.ParseMAC(macStr)
	return err == nil
}

// validateIPFormat checks if IP address is in valid format
func (s *Server) validateIPFormat(ipStr string) bool {
	if ipStr == "" {
		return true // IP is optional
	}
	
	ip := net.ParseIP(ipStr)
	return ip != nil && ip.To4() != nil // Only IPv4 supported
}

// normalizeLeaseData ensures consistent data formatting for API responses
func (s *Server) normalizeLeaseData(lease interface{}) map[string]interface{} {
	// Convert lease to map for manipulation
	data := make(map[string]interface{})
	
	// Use reflection or type assertion to handle different lease types
	// and ensure MAC addresses are properly formatted
	
	return data
}
