// ===== internal/web/server.go (Updated) =====
package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	
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
}

// TemplateData holds data for template rendering
type TemplateData struct {
    PageTitle         string
    PageBody          template.HTML    // Changed from string to template.HTML
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
}

// handleRoot handles the main page requests
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	log.Printf("Request from %s: %s", r.RemoteAddr, r.URL.String())
	
	// Handle API requests
	if apiType, exists := query["api"]; exists && len(apiType) > 0 {
		switch apiType[0] {
		case "static":
			s.handleStaticAPI(w, r)
		default:
			s.handleAPI(w, r, apiType[0])
		}
		return
	}
	
	// Handle page requests
	pageType := "splash"
	if p, exists := query["p"]; exists && len(p) > 0 {
		pageType = p[0]
	}
	
	s.handlePage(w, r, pageType)
}

// loadTemplates loads all template files
func (s *Server) loadTemplates() {
	templateFiles := map[string]string{
		"bootstrap": "bootstrap.tmpl",
		"leases":    "leases.tmpl",
		"logs":      "logs.tmpl",
		"splash":    "splash.tmpl",
		"hosts":     "hosts.tmpl",
		"static":    "static.tmpl",
	}
	
	for name, filename := range templateFiles {
		path := filepath.Join(s.cfg.HTMLDir, filename)
		
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			log.Printf("Warning: failed to load template %s: %v", filename, err)
			continue
		}
		
		s.templates[name] = tmpl
		log.Printf("Loaded template: %s", filename)
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
	
	switch pageType {
	case "Leases":
		data.PageTitle = "DHCPmon - Leases"
		content, err = s.renderTemplate("leases", data)
	case "Logs":
		data.PageTitle = "DHCPmon - Logs"
		content, err = s.renderTemplate("logs", data)
	case "Hosts":
		data.PageTitle = "DHCPmon - Hosts"
		content, err = s.renderTemplate("hosts", data)
	case "Static":
		data.PageTitle = "DHCPmon - Static DHCP"
		content, err = s.renderTemplate("static", data)
	default:
		data.PageTitle = "DHCPmon - Home"
		content, err = s.renderTemplate("splash", nil)
	}
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Convert to template.HTML to prevent escaping
	data.PageBody = template.HTML(content)
	
	// Render main template
	if tmpl, exists := s.templates["bootstrap"]; exists {
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Failed to execute bootstrap template: %v", err)
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

