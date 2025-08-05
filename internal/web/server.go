// ===== internal/web/server.go (Updated with Configurable Templates) =====
package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"os"
	"runtime"
	
	"dhcpmon/internal/config"
	"dhcpmon/internal/monitor"
	"dhcpmon/pkg/models"
)

// Server represents the HTTP server
type Server struct {
	cfg       *config.Config
	monitor   *monitor.Monitor
	templates map[string]*template.Template
	mux       *http.ServeMux
	startTime time.Time
}

// TemplateData holds data for template rendering
type TemplateData struct {
    PageTitle         string
    PageBody          template.HTML
    EnableHTTPLinks   bool
    EnableHTTPSLinks  bool
    EnableSSHLinks    bool
    EnableNetworkTags bool
    EnableEdit        bool
}

// NewServer creates a new web server
func NewServer(cfg *config.Config, mon *monitor.Monitor) *Server {
	server := &Server{
		cfg:       cfg,
		monitor:   mon,
		templates: make(map[string]*template.Template),
		mux:       http.NewServeMux(),
		startTime: time.Now(),
	}
	
	server.loadTemplates()
	server.setupRoutes()
	
	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return http.ListenAndServe(s.cfg.HTTPListen, s.mux)
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/", s.handleRoot)
	s.mux.HandleFunc("/api/static", s.handleStaticAPI)
	s.mux.HandleFunc("/api/edit", s.handleEditAPI)
}

// handleRoot handles the main page requests
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	log.Printf("Request from %s: %s", r.RemoteAddr, r.URL.String())
	
	// Handle API requests
	if apiType, exists := query["api"]; exists && len(apiType) > 0 {
		log.Printf("API request: %s", apiType[0])
		
		// Route API calls to appropriate handlers
		switch apiType[0] {
		case "static":
			s.handleStaticAPI(w, r)
		case "leases.json":
			s.handleLeasesAPI(w, r)
		case "hosts.json":
			s.handleHostsAPI(w, r)
		case "logs.json":
			s.handleLogsAPI(w, r)
		case "remove":
			s.handleRemoveAPI(w, r)
		case "edit":
			s.handleEditAPI(w, r)
		case "system.json":
			s.handleSystemAPI(w, r)
		case "version":
			s.handleVersionAPI(w, r)
		case "dhcp-status":
			s.handleDHCPStatusAPI(w, r)
		case "file-status":
			s.handleFileStatusAPI(w, r)
		case "process-info":
			s.handleProcessInfoAPI(w, r)
		case "recent-events":
			s.handleRecentEventsAPI(w, r)
		default:
			log.Printf("Unknown API type: %s", apiType[0])
			http.Error(w, `{"error":"Unknown API endpoint"}`, http.StatusNotFound)
		}
		return
	}
	
	// Handle page requests
	pageType := "Leases"
	if p, exists := query["p"]; exists && len(p) > 0 {
		pageType = p[0]
		log.Printf("Page request: %s", pageType)
	}
	
	s.handlePage(w, r, pageType)
}

// loadTemplates loads all template files using configuration
func (s *Server) loadTemplates() {
	templateMap := s.cfg.GetTemplateMap()
	
	for name, filename := range templateMap {
		path := filepath.Join(s.cfg.HTMLDir, filename)
		
		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("Warning: template file %s not found, skipping %s template", filename, name)
			continue
		}
		
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			log.Printf("Warning: failed to load template %s (%s): %v", name, filename, err)
			continue
		}
		
		s.templates[name] = tmpl
		log.Printf("Loaded template: %s -> %s", name, filename)
	}
	
	// Validate that we have the bootstrap template
	if _, exists := s.templates["bootstrap"]; !exists {
		log.Fatalf("Bootstrap template is required but not found")
	}
}

// handlePage handles page rendering
func (s *Server) handlePage(w http.ResponseWriter, r *http.Request, pageType string) {
	data := TemplateData{
		EnableHTTPLinks:   s.cfg.HTTPLinks,
		EnableHTTPSLinks:  s.cfg.HTTPSLinks,
		EnableSSHLinks:    s.cfg.SSHLinks,
		EnableNetworkTags: s.cfg.NetworkTags,
		EnableEdit:        s.cfg.Edit,
	}
	
	var content string
	var err error
	var templateName string
	
	switch pageType {
	case "Leases":
		data.PageTitle = "DHCPmon - Leases"
		templateName = "leases"
	case "Logs":
		data.PageTitle = "DHCPmon - Logs"
		templateName = "logs"
	case "Hosts":
		data.PageTitle = "DHCPmon - Hosts"
		templateName = "hosts"
	case "System":
		data.PageTitle = "DHCPmon - System"
		templateName = "system"
	case "Help":
		data.PageTitle = "DHCPmon - Help"
		templateName = "help"
	case "About":
		data.PageTitle = "DHCPmon - About"
		templateName = "about"
	default:
		// Default to leases page
		data.PageTitle = "DHCPmon - Leases"
		templateName = "leases"
	}
	
	content, err = s.renderTemplate(templateName, data)
	if err != nil {
		// If template fails, show error page
		content = fmt.Sprintf(`
			<div class="alert alert-danger">
				<h4><i class="fas fa-exclamation-triangle me-2"></i>Template Error</h4>
				<p>Failed to load template '%s': %v</p>
				<p>Please check your template configuration in dhcpmon.ini</p>
			</div>
		`, templateName, err)
	}
	
	// Convert to template.HTML to prevent escaping
	data.PageBody = template.HTML(content)
	
	// Render main template
	if tmpl, exists := s.templates["bootstrap"]; exists {
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Failed to execute bootstrap template: %v", err)
			http.Error(w, "Template execution failed", http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "Bootstrap template not found", http.StatusInternalServerError)
	}
}

// renderTemplate renders a template with given data
func (s *Server) renderTemplate(name string, data interface{}) (string, error) {
	tmpl, exists := s.templates[name]
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}
	
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// New API handlers for system information

// handleSystemAPI returns system metrics and information
func (s *Server) handleSystemAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	response := map[string]interface{}{
		"memory": map[string]interface{}{
			"used":  m.Alloc,
			"total": m.Sys,
		},
		"cpu":     0.0, // CPU usage would need OS-specific implementation
		"systemd": s.cfg.SystemD,
		"uptime":  time.Since(s.startTime).Seconds(),
	}
	
	if err := s.writeJSONResponse(w, response); err != nil {
		log.Printf("Failed to encode system JSON: %v", err)
	}
}

// handleVersionAPI returns version information
func (s *Server) handleVersionAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	// These would typically be set at build time
	response := map[string]interface{}{
		"version":   "1.0.0",
		"buildTime": s.startTime.Format(time.RFC3339),
		"gitHash":   "development",
		"goVersion": runtime.Version(),
	}
	
	if err := s.writeJSONResponse(w, response); err != nil {
		log.Printf("Failed to encode version JSON: %v", err)
	}
}

// handleDHCPStatusAPI returns DHCP service status
func (s *Server) handleDHCPStatusAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	// Check if DHCP service is running (simplified check)
	running := true
	if _, err := os.Stat(s.cfg.LeasesFile); os.IsNotExist(err) {
		running = false
	}
	
	response := map[string]interface{}{
		"running": running,
		"service": "dnsmasq", // Could be made configurable
	}
	
	if err := s.writeJSONResponse(w, response); err != nil {
		log.Printf("Failed to encode DHCP status JSON: %v", err)
	}
}

// handleFileStatusAPI returns status of monitored files
func (s *Server) handleFileStatusAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	files := []map[string]interface{}{}
	
	// Check each monitored file
	filesToCheck := map[string]string{
		"Leases File": s.cfg.LeasesFile,
		"Hosts File":  s.cfg.HostsFile,
		"Static File": s.cfg.StaticFile,
		"MAC DB":      s.cfg.MACDBFile,
	}
	
	for name, path := range filesToCheck {
		fileInfo := map[string]interface{}{
			"name": name,
			"path": path,
		}
		
		if stat, err := os.Stat(path); err == nil {
			fileInfo["exists"] = true
			fileInfo["size"] = stat.Size()
			fileInfo["modified"] = stat.ModTime()
			fileInfo["permissions"] = stat.Mode().String()
		} else {
			fileInfo["exists"] = false
			fileInfo["size"] = 0
			fileInfo["modified"] = nil
			fileInfo["permissions"] = ""
		}
		
		files = append(files, fileInfo)
	}
	
	response := map[string]interface{}{
		"files": files,
		"config": map[string]interface{}{
			"configFile":     "dhcpmon.ini",
			"configModified": s.startTime, // Simplified
			"leasesFile":     s.cfg.LeasesFile,
			"staticFile":     s.cfg.StaticFile,
		},
	}
	
	if err := s.writeJSONResponse(w, response); err != nil {
		log.Printf("Failed to encode file status JSON: %v", err)
	}
}

// handleProcessInfoAPI returns process information
func (s *Server) handleProcessInfoAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	response := map[string]interface{}{
		"pid":        os.Getpid(),
		"goVersion":  runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"gcCycles":   m.NumGC,
	}
	
	if err := s.writeJSONResponse(w, response); err != nil {
		log.Printf("Failed to encode process info JSON: %v", err)
	}
}

// handleRecentEventsAPI returns recent system events
func (s *Server) handleRecentEventsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	// This would typically pull from a real event log
	events := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-5 * time.Minute),
			"type":      "info",
			"message":   "DHCP Monitor started successfully",
		},
		{
			"timestamp": time.Now().Add(-2 * time.Minute),
			"type":      "info",
			"message":   "File watcher initialized",
		},
		{
			"timestamp": time.Now().Add(-1 * time.Minute),
			"type":      "success",
			"message":   "Configuration loaded successfully",
		},
	}
	
	response := map[string]interface{}{
		"events": events,
	}
	
	if err := s.writeJSONResponse(w, response); err != nil {
		log.Printf("Failed to encode recent events JSON: %v", err)
	}
}

// writeJSONResponse helper function to write JSON responses
func (s *Server) writeJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(data)
}

// getDHCPLeasesJSON returns DHCP leases in JSON format
func (s *Server) getDHCPLeasesJSON() []map[string]interface{} {
	leases := s.monitor.GetDHCPLeases()
	result := make([]map[string]interface{}, len(leases))
	
	for i, lease := range leases {
		result[i] = map[string]interface{}{
			"expire":  lease.Expire.Format(time.RFC3339),
			"remain":  lease.Remain.Round(time.Second).String(),
			"delta":   lease.Remain,
			"mac":     lease.MAC.String(),
			"info":    lease.Info,
			"ip":      lease.IP.String(),
			"ipSort":  s.ipToInt(lease.IP),
			"name":    lease.Name,
			"id":      lease.ID,
			"tag":     lease.Tag,
			"static":  lease.Static,
		}
	}
	
	return result
}

// getHostsJSON returns host entries in JSON format
func (s *Server) getHostsJSON() []models.HostEntry {
	return s.monitor.GetHostEntries()
}

// handleRemove handles remove requests
func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse the data parameter
	dataStr := r.FormValue("data")
	log.Printf("Remove request data: %s", dataStr)
	
	// For now, just log the request
	// TODO: Implement actual removal logic
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ReloadTemplates reloads all templates (useful for development)
func (s *Server) ReloadTemplates() {
	log.Println("Reloading templates...")
	s.loadTemplates()
}

// GetTemplateStatus returns the status of loaded templates
func (s *Server) GetTemplateStatus() map[string]bool {
	templateMap := s.cfg.GetTemplateMap()
	status := make(map[string]bool)
	
	for name := range templateMap {
		_, exists := s.templates[name]
		status[name] = exists
	}
	
	return status
}
