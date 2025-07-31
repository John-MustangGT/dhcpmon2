package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	
	"dhcpmon/internal/config"
	"dhcpmon/internal/mac"
	"dhcpmon/internal/dhcp"
	"dhcpmon/internal/web"
	"dhcpmon/internal/monitor"
)

const (
	configFile = "dhcpmon.ini"
)

var (
	sha1ver   string
	buildTime string
	repoName  string
)

func main() {
	log.Printf("%s: Build %s, Time %s", repoName, sha1ver, buildTime)
	
	// Load configuration
	cfg, err := config.New(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize MAC database
	macDB, err := mac.NewDatabase(cfg.MACDBFile, cfg.MACDBPreload)
	if err != nil {
		log.Fatalf("Failed to initialize MAC database: %v", err)
	}
	defer macDB.Close()
	
	// Initialize DHCP parser
	dhcpParser := dhcp.NewParser(macDB, cfg.StaticFile)
	
	// Initialize monitor
	monitor := monitor.New(cfg, dhcpParser)
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()
	
	// Initialize web server
	webServer := web.NewServer(cfg, monitor)
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTPListen)
		if err := webServer.Start(); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	log.Println("Shutting down...")
}
