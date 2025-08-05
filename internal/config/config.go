// ===== internal/config/config.go (Updated with HTML Templates) =====
package config

import (
	"os"
	"strconv"
	"log"
	"gopkg.in/ini.v1"
)

// HTMLTemplates holds template file mappings
type HTMLTemplates struct {
	Bootstrap string
	Leases    string
	Hosts     string
	Logs      string
	Help      string
	About     string
	System    string
}

// Config holds all application configuration
type Config struct {
	// File paths
	LeasesFile    string
	HTMLDir       string
	MACDBFile     string
	HostsFile     string
	StaticFile    string
	
	// Network settings
	HTTPListen    string
	NmapOpts      string
	
	// Binary paths
	DNSMasq       string
	Nmap          string
	
	// Feature flags
	SystemD       bool
	MACDBPreload  bool
	HTTPLinks     bool
	HTTPSLinks    bool
	SSHLinks      bool
	NetworkTags   bool
	Edit          bool
	
	// HTML Templates
	Templates     HTMLTemplates
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		LeasesFile:   "/var/lib/misc/dnsmasq.leases",
		HTMLDir:      "/app/html",
		HTTPListen:   "127.0.0.1:8067",
		DNSMasq:      "/usr/sbin/dnsmasq",
		SystemD:      false,
		MACDBFile:    "/app/macaddress.io-db.json",
		MACDBPreload: false,
		Nmap:         "/usr/bin/nmap",
		NmapOpts:     "-oG - -n -F 192.168.12.0/24",
		HostsFile:    "/var/lib/misc/hosts",
		HTTPLinks:    true,
		HTTPSLinks:   true,
		SSHLinks:     true,
		StaticFile:   "/etc/dnsmasq.d/static.conf",
		NetworkTags:  false,
		Edit:         true,
		Templates: HTMLTemplates{
			Bootstrap: "bootstrap.tmpl",
			Leases:    "leases.tmpl", 
			Hosts:     "hosts.tmpl",
			Logs:      "logs.tmpl",
			Help:      "help.tmpl",
			About:     "about.tmpl",
			System:    "system.tmpl",
		},
	}
}

// LoadFromFile loads configuration from INI file
func (c *Config) LoadFromFile(filename string) error {
	cfg, err := ini.LoadSources(ini.LoadOptions{Insensitive: true}, filename)
	if err != nil {
		log.Printf("Skipping config file %s: %s", filename, err)
		return err
	}

	// Load main section
	section := cfg.Section("")
	c.LeasesFile = section.Key("leasesfile").MustString(c.LeasesFile)
	c.HTMLDir = section.Key("htmldir").MustString(c.HTMLDir)
	c.HTTPListen = section.Key("httplisten").MustString(c.HTTPListen)
	c.DNSMasq = section.Key("dnsmasq").MustString(c.DNSMasq)
	c.SystemD = section.Key("systemd").MustBool(c.SystemD)
	c.MACDBFile = section.Key("macdbfile").MustString(c.MACDBFile)
	c.MACDBPreload = section.Key("macdbpreload").MustBool(c.MACDBPreload)
	c.Nmap = section.Key("nmap").MustString(c.Nmap)
	c.NmapOpts = section.Key("nmapopts").MustString(c.NmapOpts)
	c.HostsFile = section.Key("hostsfile").MustString(c.HostsFile)
	c.HTTPLinks = section.Key("httplinks").MustBool(c.HTTPLinks)
	c.HTTPSLinks = section.Key("httpslinks").MustBool(c.HTTPSLinks)
	c.SSHLinks = section.Key("sshlinks").MustBool(c.SSHLinks)
	c.StaticFile = section.Key("staticfile").MustString(c.StaticFile)
	c.NetworkTags = section.Key("networktags").MustBool(c.NetworkTags)
	c.Edit = section.Key("edit").MustBool(c.Edit)

	// Load HTML templates section
	if htmlSection, err := cfg.GetSection("html"); err == nil {
		c.Templates.Bootstrap = htmlSection.Key("bootstrap").MustString(c.Templates.Bootstrap)
		c.Templates.Leases = htmlSection.Key("leases").MustString(c.Templates.Leases)
		c.Templates.Hosts = htmlSection.Key("hosts").MustString(c.Templates.Hosts)
		c.Templates.Logs = htmlSection.Key("logs").MustString(c.Templates.Logs)
		c.Templates.Help = htmlSection.Key("help").MustString(c.Templates.Help)
		c.Templates.About = htmlSection.Key("about").MustString(c.Templates.About)
		c.Templates.System = htmlSection.Key("system").MustString(c.Templates.System)
	}

	return nil
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("LEASESFILE"); v != "" {
		c.LeasesFile = v
	}
	if v := os.Getenv("HTMLDIR"); v != "" {
		c.HTMLDir = v
	}
	if v := os.Getenv("HTTPLISTEN"); v != "" {
		c.HTTPListen = v
	}
	if v := os.Getenv("DNSMASQ"); v != "" {
		c.DNSMasq = v
	}
	if v := os.Getenv("SYSTEMD"); v != "" {
		c.SystemD, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("MACDBFILE"); v != "" {
		c.MACDBFile = v
	}
	if v := os.Getenv("MACDBPRELOAD"); v != "" {
		c.MACDBPreload, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("NMAP"); v != "" {
		c.Nmap = v
	}
	if v := os.Getenv("NMAPOPTS"); v != "" {
		c.NmapOpts = v
	}
	if v := os.Getenv("HOSTSFILE"); v != "" {
		c.HostsFile = v
	}
	if v := os.Getenv("HTTPLINKS"); v != "" {
		c.HTTPLinks, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("HTTPSLINKS"); v != "" {
		c.HTTPSLinks, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("SSHLINKS"); v != "" {
		c.SSHLinks, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("STATICFILE"); v != "" {
		c.StaticFile = v
	}
	if v := os.Getenv("NETWORKTAGS"); v != "" {
		c.NetworkTags, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("EDIT"); v != "" {
		c.Edit, _ = strconv.ParseBool(v)
	}
	
	// HTML template environment variables
	if v := os.Getenv("HTML_BOOTSTRAP"); v != "" {
		c.Templates.Bootstrap = v
	}
	if v := os.Getenv("HTML_LEASES"); v != "" {
		c.Templates.Leases = v
	}
	if v := os.Getenv("HTML_HOSTS"); v != "" {
		c.Templates.Hosts = v
	}
	if v := os.Getenv("HTML_LOGS"); v != "" {
		c.Templates.Logs = v
	}
	if v := os.Getenv("HTML_HELP"); v != "" {
		c.Templates.Help = v
	}
	if v := os.Getenv("HTML_ABOUT"); v != "" {
		c.Templates.About = v
	}
	if v := os.Getenv("HTML_SYSTEM"); v != "" {
		c.Templates.System = v
	}
}

// New creates a new configuration instance
func New(configFile string) (*Config, error) {
	cfg := DefaultConfig()
	
	// Load from file first
	cfg.LoadFromFile(configFile)
	
	// Override with environment variables
	cfg.LoadFromEnv()
	
	return cfg, nil
}

// GetTemplateMap returns a map of template names to filenames
func (c *Config) GetTemplateMap() map[string]string {
	return map[string]string{
		"bootstrap": c.Templates.Bootstrap,
		"leases":    c.Templates.Leases,
		"hosts":     c.Templates.Hosts,
		"logs":      c.Templates.Logs,
		"help":      c.Templates.Help,
		"about":     c.Templates.About,
		"system":    c.Templates.System,
	}
}

// ValidateTemplates checks if all template files exist
func (c *Config) ValidateTemplates() []string {
	var missing []string
	templateMap := c.GetTemplateMap()
	
	for name, filename := range templateMap {
		fullPath := c.HTMLDir + "/" + filename
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			missing = append(missing, name+" ("+filename+")")
		}
	}
	
	return missing
}