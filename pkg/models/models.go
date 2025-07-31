// ===== pkg/models/models.go =====
package models

import (
	"net"
	"time"
)

// DHCPLease represents a DHCP lease entry
type DHCPLease struct {
	Expire time.Time        `json:"expire"`
	Remain time.Duration    `json:"remain"`
	MAC    net.HardwareAddr `json:"mac"`
	Info   *OUIEntry        `json:"info"`
	IP     net.IP           `json:"ip"`
	Name   string           `json:"name"`
	ID     string           `json:"id"`
	Tag    string           `json:"tag"`
	Static bool             `json:"static"`
}

// OUIEntry represents MAC address vendor information
type OUIEntry struct {
	OUI         string `json:"oui"`
	Private     bool   `json:"isPrivate"`
	Company     string `json:"companyName"`
	Address     string `json:"companyAddress"`
	CountryCode string `json:"countryCode"`
	BlockSize   string `json:"assignmentBlockSize"`
	Created     string `json:"dateCreated"`
	Updated     string `json:"dateUpdated"`
}

// HostEntry represents a hosts file entry
type HostEntry struct {
	IP    string   `json:"ip"`
	Name  string   `json:"name"`
	Alias []string `json:"alias"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time `json:"when"`
	UnixTime  int64     `json:"utime"`
	Channel   string    `json:"channel"`
	Message   string    `json:"message"`
}

