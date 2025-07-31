# DHCP Monitor

A modern, refactored DHCP monitoring tool with a clean web interface for tracking DHCP leases, static assignments, and network activity.

## Features

- **Real-time DHCP lease monitoring** - Track active and expired leases
- **Static DHCP reservations** - Monitor configured static assignments
- **MAC address vendor lookup** - Identify device manufacturers
- **Hosts file integration** - Display configured host entries
- **Live log monitoring** - View dnsmasq logs in real-time
- **Responsive web interface** - Clean, mobile-friendly UI
- **File system monitoring** - Automatic updates when files change
- **Systemd integration** - Support for systemd journal logs

## Architecture

The application has been completely refactored with a clean, modular architecture:

```
├── cmd/dhcpmon/           # Application entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── dhcp/              # DHCP lease parsing
│   ├── hosts/             # Hosts file parsing
│   ├── mac/               # MAC address database
│   ├── monitor/           # File monitoring and data management
│   ├── web/               # HTTP server and API
│   └── logs/              # Log collection and management
├── pkg/
│   ├── models/            # Shared data structures
│   └── utils/             # Utility functions
└── html/                  # Web templates
```

## Configuration

Configuration can be provided via INI file (`dhcpmon.ini`) or environment variables:

### INI File Example
```ini
leasesfile=/var/lib/misc/dnsmasq.leases
htmldir=/app/html
httplisten=127.0.0.1:8067
dnsmasq=/usr/sbin/dnsmasq
systemd=false
macdbfile=/app/macaddress.io-db.json
macdbpreload=false
nmap=/usr/bin/nmap
nmapopts=-oG - -n -F 192.168.12.0/24
hostsfile=/var/lib/misc/hosts
httplinks=true
httpslinks=true
sshlinks=true
staticfile=/etc/dnsmasq.d/static.conf
networktags=false
edit=true
```

### Environment Variables
All configuration options can be set via environment variables by converting to uppercase:
- `LEASESFILE`
- `HTMLDIR`
- `HTTPLISTEN`
- etc.

## Building

### Prerequisites
- Go 1.21 or later
- Make (optional, for using Makefile)

### Build Commands

```bash
# Build the application
make build

# Run tests
make test

# Build for multiple platforms
make build-all

# Build Docker image
make docker-build

# Run with Docker
make docker-run
```

### Manual Build
```bash
go mod download
go build -o dhcpmon cmd/dhcpmon/main.go
```

## Running

### Standalone
```bash
./dhcpmon
```

### Docker
```bash
docker run --rm -p 8067:8067 \
  -v /var/lib/misc:/var/lib/misc:ro \
  -v /etc/dnsmasq.d:/etc/dnsmasq.d:ro \
  dhcpmon:latest
```

### Systemd Service
```ini
[Unit]
Description=DHCP Monitor
After=network.target

[Service]
Type=simple
User=dhcpmon
WorkingDirectory=/opt/dhcpmon
ExecStart=/opt/dhcpmon/dhcpmon
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## API Endpoints

The application provides REST API endpoints:

- `GET /?api=leases.json` - Get DHCP leases
- `GET /?api=hosts.json` - Get hosts file entries  
- `GET /?api=logs.json` - Get log entries
- `POST /?api=remove` - Remove entry (with JSON data)
- `POST /?api=edit` - Edit entry (with JSON data)

## Web Interface

Access the web interface at `http://localhost:8067`

Pages available:
- **Leases** - Active DHCP leases with device information
- **Hosts** - Static host entries from hosts file
- **Logs** - Real-time log monitoring
- **Config** - Configuration overview (planned)

## Key Improvements in Refactor

1. **Modular Architecture** - Separated concerns into distinct packages
2. **Better Error Handling** - Comprehensive error handling throughout
3. **Concurrent Safety** - Proper mutex usage for shared data
4. **Resource Management** - Proper cleanup of resources and goroutines
5. **Configuration Management** - Centralized, type-safe configuration
6. **Testing Ready** - Structure allows for easy unit testing
7. **Graceful Shutdown** - Proper signal handling and cleanup
8. **Logging** - Structured logging throughout the application
9. **Type Safety** - Strong typing with clear interfaces
10. **Documentation** - Well-documented code with clear responsibilities

## Dependencies

- `github.com/fsnotify/fsnotify` - File system notifications
- `gopkg.in/ini.v1` - INI file parsing

## License

This project is licensed under the MIT License.
