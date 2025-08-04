// ===== internal/web/templates.go =====
package web

import (
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"strings"
)

// TemplateManager handles template loading and rendering
type TemplateManager struct {
	templates map[string]*template.Template
	htmlDir   string
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(htmlDir string) *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*template.Template),
		htmlDir:   htmlDir,
	}
}

// LoadTemplates loads all templates from the HTML directory
func (tm *TemplateManager) LoadTemplates() error {
	templateFiles := map[string]string{
		"bootstrap": "bootstrap.tmpl",
		"header": "header.tmpl",
		"footer": "footer.tmpl",
		"leases":    "leases.tmpl",
		"logs":      "logs.tmpl",
		"splash":    "splash.tmpl",
		"hosts":     "hosts.tmpl",
	}
	
	for name, filename := range templateFiles {
		path := filepath.Join(tm.htmlDir, filename)
		
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			log.Printf("Warning: failed to load template %s: %v", filename, err)
			continue
		}
		
		tm.templates[name] = tmpl
		log.Printf("Loaded template: %s", filename)
	}
	
	return nil
}

// Render renders a template with the given data
func (tm *TemplateManager) Render(name string, data interface{}) (string, error) {
	tmpl, exists := tm.templates[name]
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}
	
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	
	return buf.String(), nil
}

// HasTemplate checks if a template exists
func (tm *TemplateManager) HasTemplate(name string) bool {
	_, exists := tm.templates[name]
	return exists
}

