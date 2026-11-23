package main

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	ip2region "github.com/lionsoul2014/ip2region/binding/golang/service"
)

const defaultGeoIPSearchers = 16

var (
	ErrGeoIPClosed        = errors.New("GeoIP service is closed")
	ErrGeoIPNotConfigured = errors.New("no IP2Region database configured")
)

// GeoIPConfig configures the local IP2Region databases.
type GeoIPConfig struct {
	IPv4DBPath string
	IPv6DBPath string
	Searchers  int
}

// GeoIPInfo contains the raw location information returned by IP2Region.
type GeoIPInfo struct {
	IP          string
	Country     string
	Province    string
	City        string
	ISP         string
	CountryCode string
	Found       bool
}

// GeoIP provides concurrent-safe IPv4 and IPv6 lookup.
type GeoIP struct {
	mu      sync.RWMutex
	service *ip2region.Ip2Region
	closed  bool
}

// NewGeoIP initializes the configured IP2Region databases.
//
// At least one database path must be provided. Searchers defaults to 16
// when configured as zero. Negative values are rejected.
func NewGeoIP(config GeoIPConfig) (*GeoIP, error) {
	v4Path := strings.TrimSpace(config.IPv4DBPath)
	v6Path := strings.TrimSpace(config.IPv6DBPath)

	if v4Path == "" && v6Path == "" {
		return nil, ErrGeoIPNotConfigured
	}

	searchers := config.Searchers
	if searchers == 0 {
		searchers = defaultGeoIPSearchers
	}
	if searchers < 1 {
		return nil, fmt.Errorf("invalid GeoIP searcher count: %d", searchers)
	}

	var v4Config *ip2region.Config
	var v6Config *ip2region.Config
	var err error

	if v4Path != "" {
		v4Config, err = ip2region.NewV4Config(
			ip2region.VIndexCache,
			v4Path,
			searchers,
		)
		if err != nil {
			return nil, fmt.Errorf("initialize IPv4 IP2Region database %q: %w", v4Path, err)
		}
	}

	if v6Path != "" {
		v6Config, err = ip2region.NewV6Config(
			ip2region.VIndexCache,
			v6Path,
			searchers,
		)
		if err != nil {
			return nil, fmt.Errorf("initialize IPv6 IP2Region database %q: %w", v6Path, err)
		}
	}

	svc, err := ip2region.NewIp2Region(v4Config, v6Config)
	if err != nil {
		return nil, fmt.Errorf("create IP2Region service: %w", err)
	}

	return &GeoIP{service: svc}, nil
}

// Search looks up an IPv4 or IPv6 address.
//
// An IP address not found in the database returns Found=false and no error.
// The ISP field is the original text from IP2Region; normalization belongs
// to the filtering layer.
func (g *GeoIP) Search(ip string) (GeoIPInfo, error) {
	if g == nil {
		return GeoIPInfo{}, errors.New("GeoIP service is nil")
	}

	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return GeoIPInfo{}, fmt.Errorf("invalid IP address %q", ip)
	}

	// Normalize IPv4-mapped IPv6 addresses to IPv4.
	if ipv4 := parsedIP.To4(); ipv4 != nil {
		parsedIP = ipv4
	} else {
		parsedIP = parsedIP.To16()
	}

	if parsedIP == nil {
		return GeoIPInfo{}, fmt.Errorf("invalid IP address %q", ip)
	}

	normalizedIP := parsedIP.String()

	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.closed || g.service == nil {
		return GeoIPInfo{}, ErrGeoIPClosed
	}

	region, err := g.service.Search(normalizedIP)
	if err != nil {
		return GeoIPInfo{}, fmt.Errorf("lookup IP %s: %w", normalizedIP, err)
	}

	info := GeoIPInfo{
		IP: normalizedIP,
	}

	if region == "" {
		return info, nil
	}

	// IP2Region format:
	// Country|Province|City|ISP|iso-alpha2-code
	fields := strings.Split(region, "|")
	if len(fields) != 5 {
		return GeoIPInfo{}, fmt.Errorf(
			"unexpected IP2Region result for %s: expected 5 fields, got %d",
			normalizedIP,
			len(fields),
		)
	}

	info.Country = fields[0]
	info.Province = fields[1]
	info.City = fields[2]
	info.ISP = fields[3]
	info.CountryCode = fields[4]
	info.Found = true

	return info, nil
}

// Close releases the IP2Region service. It is safe to call more than once.
// It waits for in-flight Search calls to finish.
func (g *GeoIP) Close() {
	if g == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed {
		return
	}

	g.closed = true
	if g.service != nil {
		g.service.Close()
		g.service = nil
	}
}
