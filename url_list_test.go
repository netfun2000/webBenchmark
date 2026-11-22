package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestParseURLList(t *testing.T) {
	input := "\uFEFFhttps://example.com/a\n" +
		"\n" +
		"https://example.com/b\n" +
		"not-a-url\n" +
		"https://example.com/a\n"

	list, err := parseURLListString(input)
	if err != nil {
		t.Fatalf("parseURLListString returned error: %v", err)
	}

	wantURLs := []string{
		"https://example.com/a",
		"https://example.com/b",
		"https://example.com/a",
	}
	if !reflect.DeepEqual(list.URLs, wantURLs) {
		t.Fatalf("URLs = %#v, want %#v", list.URLs, wantURLs)
	}

	if len(list.Invalid) != 1 {
		t.Fatalf("invalid count = %d, want 1", len(list.Invalid))
	}
	if list.Invalid[0].Line != 4 {
		t.Fatalf("invalid line = %d, want 4", list.Invalid[0].Line)
	}
	if list.Invalid[0].Value != "not-a-url" {
		t.Fatalf("invalid value = %q, want not-a-url", list.Invalid[0].Value)
	}
}

func parseURLListString(input string) (*urlList, error) {
	return parseURLList(strings.NewReader(input))
}

func TestLoadURLListFromLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "urls.txt")

	content := "https://example.com/1\nhttps://example.com/2\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	list, err := loadURLList(context.Background(), path)
	if err != nil {
		t.Fatalf("loadURLList returned error: %v", err)
	}

	want := []string{
		"https://example.com/1",
		"https://example.com/2",
	}
	if !reflect.DeepEqual(list.URLs, want) {
		t.Fatalf("URLs = %#v, want %#v", list.URLs, want)
	}
}

func TestLoadURLListFromRemoteHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		fmt.Fprintln(w, "https://example.com/a")
		fmt.Fprintln(w, "https://example.com/b")
	}))
	defer server.Close()

	list, err := loadURLList(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("loadURLList returned error: %v", err)
	}

	want := []string{
		"https://example.com/a",
		"https://example.com/b",
	}
	if !reflect.DeepEqual(list.URLs, want) {
		t.Fatalf("URLs = %#v, want %#v", list.URLs, want)
	}
}

func TestLoadURLListRejectsHTTPErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := loadURLList(context.Background(), server.URL); err == nil {
		t.Fatal("expected an error for HTTP 404")
	}
}

func TestLoadURLListRejectsEmptyValidList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.txt")

	if err := os.WriteFile(path, []byte("\nnot-a-url\nftp://example.com\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	list, err := loadURLList(context.Background(), path)
	if err == nil {
		t.Fatal("expected an error when there are no valid URLs")
	}
	if list == nil {
		t.Fatal("expected parsed list to be returned with the error")
	}
	if len(list.URLs) != 0 {
		t.Fatalf("valid URL count = %d, want 0", len(list.URLs))
	}
	if len(list.Invalid) != 2 {
		t.Fatalf("invalid count = %d, want 2", len(list.Invalid))
	}
}

func TestURLListAllocatorSingleRound(t *testing.T) {
	urls := []string{"https://example.com/1", "https://example.com/2"}

	allocator, err := newURLListAllocator(urls, false)
	if err != nil {
		t.Fatalf("newURLListAllocator: %v", err)
	}

	for _, want := range urls {
		got, ok := allocator.Next()
		if !ok || got != want {
			t.Fatalf("Next() = (%q, %v), want (%q, true)", got, ok, want)
		}
	}

	if got, ok := allocator.Next(); ok {
		t.Fatalf("Next() after end = (%q, true), want empty and false", got)
	}
}

func TestURLListAllocatorLoop(t *testing.T) {
	urls := []string{"https://example.com/1", "https://example.com/2"}

	allocator, err := newURLListAllocator(urls, true)
	if err != nil {
		t.Fatalf("newURLListAllocator: %v", err)
	}

	want := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/1",
	}

	for i, expected := range want {
		got, ok := allocator.Next()
		if !ok || got != expected {
			t.Fatalf("Next() #%d = (%q, %v), want (%q, true)", i, got, ok, expected)
		}
	}
}

func TestURLListAllocatorConcurrent(t *testing.T) {
	const workerCount = 8
	const urlsPerWorker = 100

	urls := make([]string, urlsPerWorker)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.com/%d", i)
	}

	allocator, err := newURLListAllocator(urls, false)
	if err != nil {
		t.Fatalf("newURLListAllocator: %v", err)
	}

	var wg sync.WaitGroup
	var resultMu sync.Mutex
	results := make([]string, 0, urlsPerWorker)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				value, ok := allocator.Next()
				if !ok {
					return
				}

				resultMu.Lock()
				results = append(results, value)
				resultMu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(results) != urlsPerWorker {
		t.Fatalf("received %d URLs, want %d", len(results), urlsPerWorker)
	}

	seen := make(map[string]int)
	for _, value := range results {
		seen[value]++
	}
	for _, value := range urls {
		if seen[value] != 1 {
			t.Errorf("%q received %d times, want exactly once", value, seen[value])
		}
	}
}

func TestNewURLListAllocatorRejectsEmptyList(t *testing.T) {
	if _, err := newURLListAllocator(nil, false); err == nil {
		t.Fatal("expected an error for an empty URL list")
	}
}

func TestParseURLListRejectsOversizedLine(t *testing.T) {
	input := strings.Repeat("a", urlListMaxLine+1)

	_, err := parseURLList(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected an error for an oversized line")
	}
}

func TestLoadURLListRejectsOversizedRemoteResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, strings.Repeat("x", urlListMaxBytes+1))
	}))
	defer server.Close()

	_, err := loadURLList(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected an error for an oversized remote response")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadURLListHonorsCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := loadURLList(ctx, server.URL)
	if err == nil {
		t.Fatal("expected an error when the request context is canceled")
	}
}
