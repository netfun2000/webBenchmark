package main

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func testGeoIPDBPaths() (string, string) {
	return filepath.Join("data", "ip2region", "ip2region_v4.xdb"),
		filepath.Join("data", "ip2region", "ip2region_v6.xdb")
}

func newTestGeoIP(t *testing.T) *GeoIP {
	t.Helper()

	v4Path, v6Path := testGeoIPDBPaths()

	geo, err := NewGeoIP(GeoIPConfig{
		IPv4DBPath: v4Path,
		IPv6DBPath: v6Path,
		Searchers:  4,
	})
	if err != nil {
		t.Skipf("IP2Region test databases unavailable: %v", err)
	}

	t.Cleanup(geo.Close)
	return geo
}

func TestNewGeoIPRequiresDatabase(t *testing.T) {
	_, err := NewGeoIP(GeoIPConfig{})
	if !errors.Is(err, ErrGeoIPNotConfigured) {
		t.Fatalf("expected ErrGeoIPNotConfigured, got %v", err)
	}
}

func TestNewGeoIPRejectsNegativeSearchers(t *testing.T) {
	_, err := NewGeoIP(GeoIPConfig{
		IPv4DBPath: "unused.xdb",
		Searchers:  -1,
	})
	if err == nil {
		t.Fatal("expected error for negative searcher count")
	}
}

func TestNewGeoIPRejectsMissingDatabase(t *testing.T) {
	_, err := NewGeoIP(GeoIPConfig{
		IPv4DBPath: filepath.Join(t.TempDir(), "missing-v4.xdb"),
	})
	if err == nil {
		t.Fatal("expected error for missing database")
	}
}

func TestGeoIPSearchRejectsInvalidIP(t *testing.T) {
	geo := newTestGeoIP(t)

	_, err := geo.Search("not-an-ip")
	if err == nil {
		t.Fatal("expected invalid IP error")
	}
}

func TestGeoIPSearchIPv4(t *testing.T) {
	geo := newTestGeoIP(t)

	info, err := geo.Search("114.114.114.114")
	if err != nil {
		t.Fatalf("IPv4 lookup failed: %v", err)
	}
	if info.IP != "114.114.114.114" {
		t.Fatalf("unexpected normalized IP: %q", info.IP)
	}
	if info.Found {
		t.Logf("IPv4 result: country=%q province=%q city=%q ISP=%q code=%q",
			info.Country, info.Province, info.City, info.ISP, info.CountryCode)
	} else {
		t.Log("IPv4 address was not found in the database")
	}
}

func TestGeoIPSearchIPv6(t *testing.T) {
	geo := newTestGeoIP(t)

	info, err := geo.Search("2001:4860:4860::8888")
	if err != nil {
		t.Fatalf("IPv6 lookup failed: %v", err)
	}
	if info.IP != "2001:4860:4860::8888" {
		t.Fatalf("unexpected normalized IP: %q", info.IP)
	}
	if info.Found {
		t.Logf("IPv6 result: country=%q province=%q city=%q ISP=%q code=%q",
			info.Country, info.Province, info.City, info.ISP, info.CountryCode)
	} else {
		t.Log("IPv6 address was not found in the database")
	}
}

func TestGeoIPSearchIPv4MappedIPv6(t *testing.T) {
	geo := newTestGeoIP(t)

	info, err := geo.Search("::ffff:114.114.114.114")
	if err != nil {
		t.Fatalf("IPv4-mapped IPv6 lookup failed: %v", err)
	}
	if info.IP != "114.114.114.114" {
		t.Fatalf("expected IPv4 normalization, got %q", info.IP)
	}
}

func TestGeoIPConcurrentSearch(t *testing.T) {
	geo := newTestGeoIP(t)

	const workers = 32

	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if _, err := geo.Search("114.114.114.114"); err != nil {
				errs <- err
			}
			if _, err := geo.Search("2001:4860:4860::8888"); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent lookup failed: %v", err)
	}
}

func TestGeoIPClose(t *testing.T) {
	geo := newTestGeoIP(t)

	geo.Close()
	geo.Close()

	_, err := geo.Search("114.114.114.114")
	if !errors.Is(err, ErrGeoIPClosed) {
		t.Fatalf("expected ErrGeoIPClosed, got %v", err)
	}
}
