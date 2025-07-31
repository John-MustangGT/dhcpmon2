// ===== internal/config/config.go =====
package config

import (
	"os"
	"strconv"
	"log"
	"gopkg.in/ini.v1"
)

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
	}
}

// LoadFromFile loads configuration from INI file
func (c *Config) LoadFromFile(filename string) error {
	cfg, err := ini.LoadSources(ini.LoadOptions{Insensitive: true}, filename)
	if err != nil {
		log.Printf("Skipping config file %s: %s", filename, err)
		return err
	}

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

