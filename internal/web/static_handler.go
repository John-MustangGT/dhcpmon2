// ===== internal/web/static_handlers.go =====
package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	
	"dhcpmon/pkg/models"
)

// StaticDHCPRequest represents API requests for static DHCP management
type StaticDHCPRequest struct {
	Action string                     `json:"action"`
	ID     string                     `json:"id,omitempty"`
	Entry  models.StaticDHCPEntry     `json:"entry,omitempty"`
	Filter map[string]string          `json:"filter,omitempty"`
}

// StaticDHCPResponse represents API responses for static DHCP management
type StaticDHCPResponse struct {
	Success bool                       `json:"success"`
	Message string                     `json:"message,omitempty"`
	Data    interface{}                `json:"data,omitempty"`
	Errors  []string                   `json:"errors,omitempty"`
}

// handleStaticAPI handles static DHCP configuration API requests
func (s *Server) handleStaticAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	if r.Method == http.MethodGet {
		s.handleStaticGet(w, r)
		return
	}
	
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req StaticDHCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}
	
	switch req.Action {
	case "list":
		s.handleStaticList(w, r, req)
	case "get":
		s.handleStaticGetOne(w, r, req)
	case "add":
		s.handleStaticAdd(w, r, req)
	case "update":
		s.handleStaticUpdate(w, r, req)
	case "delete":
		s.handleStaticDelete(w, r, req)
	case "enable":
		s.handleStaticEnable(w, r, req)
	case "disable":
		s.handleStaticDisable(w, r, req)
	case "validate":
		s.handleStaticValidate(w, r, req)
	case "save":
		s.handleStaticSave(w, r, req)
	case "reload":
		s.handleStaticReload(w, r, req)
	default:
		s.writeErrorResponse(w, "Unknown action", http.StatusBadRequest)
	}
}

// handleStaticGet handles GET requests for static DHCP data
func (s *Server) handleStaticGet(w http.ResponseWriter, r *http.Request) {
	entries := s.monitor.GetStaticEntries()
	
	response := StaticDHCPResponse{
		Success: true,
		Data:    entries,
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleStaticList handles list requests
func (s *Server) handleStaticList(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	entries := s.monitor.GetStaticEntries()
	
	// Apply filters if provided
	if req.Filter != nil {
		entries = s.filterStaticEntries(entries, req.Filter)
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Data:    entries,
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleStaticGetOne handles get single entry requests
func (s *Server) handleStaticGetOne(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if req.ID == "" {
		s.writeErrorResponse(w, "ID is required", http.StatusBadRequest)
		return
	}
	
	entry, err := s.monitor.GetStaticEntryByID(req.ID)
	if err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Data:    entry,
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleStaticAdd handles add entry requests
func (s *Server) handleStaticAdd(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if err := s.monitor.AddStaticEntry(req.Entry); err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Message: "Static DHCP entry added successfully",
	}
	
	json.NewEncoder(w).Encode(response)
	log.Printf("Added static DHCP entry: MAC=%s, IP=%s, Hostname=%s", 
		req.Entry.MAC.String(), req.Entry.IP.String(), req.Entry.Hostname)
}

// handleStaticUpdate handles update entry requests
func (s *Server) handleStaticUpdate(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if req.ID == "" {
		s.writeErrorResponse(w, "ID is required", http.StatusBadRequest)
		return
	}
	
	if err := s.monitor.UpdateStaticEntry(req.ID, req.Entry); err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Message: "Static DHCP entry updated successfully",
	}
	
	json.NewEncoder(w).Encode(response)
	log.Printf("Updated static DHCP entry: ID=%s", req.ID)
}

// handleStaticDelete handles delete entry requests
func (s *Server) handleStaticDelete(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if req.ID == "" {
		s.writeErrorResponse(w, "ID is required", http.StatusBadRequest)
		return
	}
	
	if err := s.monitor.DeleteStaticEntry(req.ID); err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Message: "Static DHCP entry deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
	log.Printf("Deleted static DHCP entry: ID=%s", req.ID)
}

// handleStaticEnable handles enable entry requests
func (s *Server) handleStaticEnable(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if req.ID == "" {
		s.writeErrorResponse(w, "ID is required", http.StatusBadRequest)
		return
	}
	
	if err := s.monitor.EnableStaticEntry(req.ID); err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Message: "Static DHCP entry enabled successfully",
	}
	
	json.NewEncoder(w).Encode(response)
	log.Printf("Enabled static DHCP entry: ID=%s", req.ID)
}

// handleStaticDisable handles disable entry requests
func (s *Server) handleStaticDisable(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if req.ID == "" {
		s.writeErrorResponse(w, "ID is required", http.StatusBadRequest)
		return
	}
	
	if err := s.monitor.DisableStaticEntry(req.ID); err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Message: "Static DHCP entry disabled successfully",
	}
	
	json.NewEncoder(w).Encode(response)
	log.Printf("Disabled static DHCP entry: ID=%s", req.ID)
}

// handleStaticValidate handles validate configuration requests
func (s *Server) handleStaticValidate(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	errors := s.monitor.ValidateStaticEntries()
	
	response := StaticDHCPResponse{
		Success: len(errors) == 0,
		Message: fmt.Sprintf("Validation completed with %d errors", len(errors)),
	}
	
	if len(errors) > 0 {
		errorStrings := make([]string, len(errors))
		for i, err := range errors {
			errorStrings[i] = err.Error()
		}
		response.Errors = errorStrings
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleStaticSave handles save configuration requests
func (s *Server) handleStaticSave(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if err := s.monitor.SaveStaticEntries(); err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Message: "Static DHCP configuration saved successfully",
	}
	
	json.NewEncoder(w).Encode(response)
	log.Printf("Saved static DHCP configuration to file")
}

// handleStaticReload handles reload configuration requests
func (s *Server) handleStaticReload(w http.ResponseWriter, r *http.Request, req StaticDHCPRequest) {
	if err := s.monitor.ReloadStaticEntries(); err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := StaticDHCPResponse{
		Success: true,
		Message: "Static DHCP configuration reloaded successfully",
	}
	
	json.NewEncoder(w).Encode(response)
	log.Printf("Reloaded static DHCP configuration from file")
}

// filterStaticEntries applies filters to static entries
func (s *Server) filterStaticEntries(entries []models.StaticDHCPEntry, filters map[string]string) []models.StaticDHCPEntry {
	var filtered []models.StaticDHCPEntry
	
	for _, entry := range entries {
		match := true
		
		for key, value := range filters {
			switch key {
			case "enabled":
				if (value == "true") != entry.Enabled {
					match = false
				}
			case "mac":
				if !strings.Contains(strings.ToLower(entry.MAC.String()), strings.ToLower(value)) {
					match = false
				}
			case "ip":
				if entry.IP == nil || !strings.Contains(entry.IP.String(), value) {
					match = false
				}
			case "hostname":
				if !strings.Contains(strings.ToLower(entry.Hostname), strings.ToLower(value)) {
					match = false
				}
			case "tag":
				if !strings.Contains(strings.ToLower(entry.Tag), strings.ToLower(value)) {
					match = false
				}
			}
			
			if !match {
				break
			}
		}
		
		if match {
			filtered = append(filtered, entry)
		}
	}
	
	return filtered
}

// writeErrorResponse writes a JSON error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := StaticDHCPResponse{
		Success: false,
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}
