package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	urlListMaxBytes = 16 << 20 // 远程列表最大 16 MiB
	urlListMaxLine  = 1 << 20  // 单行最大 1 MiB
	urlListTimeout  = 30 * time.Second
)

// urlListInvalid 保存 URL 列表中的无效行。
type urlListInvalid struct {
	Line   int
	Value  string
	Reason string
}

// urlList 保存解析后的 URL 和无效行。
// URLs 保留文件原始顺序及重复项。
type urlList struct {
	URLs    []string
	Invalid []urlListInvalid
}

// loadURLList 从本地文件或 HTTP/HTTPS 地址加载 URL 列表。
//
// source 以 http:// 或 https:// 开头时按远程地址处理，
// 否则按本地文件路径处理。
//
// 如果存在无效行但仍有有效 URL，会返回列表且不返回错误。
// 如果没有任何有效 URL，会返回列表和错误，调用者仍可查看无效行。
func loadURLList(ctx context.Context, source string) (*urlList, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("URL list source is empty")
	}

	var reader io.Reader

	lowerSource := strings.ToLower(source)
	if strings.HasPrefix(lowerSource, "http://") ||
		strings.HasPrefix(lowerSource, "https://") {
		if err := validateTarget(source); err != nil {
			return nil, fmt.Errorf("invalid remote URL list address: %w", err)
		}

		data, err := fetchRemoteURLList(ctx, source)
		if err != nil {
			return nil, err
		}
		reader = strings.NewReader(string(data))
	} else {
		file, err := os.Open(source)
		if err != nil {
			return nil, fmt.Errorf("open URL list file %q: %w", source, err)
		}
		defer file.Close()
		reader = file
	}

	list, err := parseURLList(reader)
	if err != nil {
		return list, err
	}
	if len(list.URLs) == 0 {
		return list, fmt.Errorf(
			"URL list contains no valid URLs (invalid lines: %d)",
			len(list.Invalid),
		)
	}

	return list, nil
}

// fetchRemoteURLList 获取远程 URL 列表。
// 限制请求时间和响应体大小，避免异常响应占用过多资源。
func fetchRemoteURLList(ctx context.Context, source string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, urlListTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("create URL list request: %w", err)
	}

	client := &http.Client{
		Timeout: urlListTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download remote URL list %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf(
			"download remote URL list %q: unexpected HTTP status %s",
			source,
			resp.Status,
		)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, urlListMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read remote URL list %q: %w", source, err)
	}
	if len(data) > urlListMaxBytes {
		return nil, fmt.Errorf(
			"remote URL list exceeds maximum size of %d bytes",
			urlListMaxBytes,
		)
	}

	return data, nil
}

// parseURLList 解析 URL 列表。
// 空行跳过；无效 URL 记录行号；重复 URL 保留。
func parseURLList(reader io.Reader) (*urlList, error) {
	list := &urlList{}
	scanner := bufio.NewScanner(reader)

	// Scanner 默认最大 token 约 64 KiB，这里提高到 1 MiB。
	scanner.Buffer(make([]byte, 64*1024), urlListMaxLine)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// 兼容 UTF-8 BOM，仅处理文件第一行。
		if lineNumber == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
			line = strings.TrimSpace(line)
		}

		if line == "" {
			continue
		}

		if err := validateTarget(line); err != nil {
			list.Invalid = append(list.Invalid, urlListInvalid{
				Line:   lineNumber,
				Value:  line,
				Reason: err.Error(),
			})
			continue
		}

		list.URLs = append(list.URLs, line)
	}

	if err := scanner.Err(); err != nil {
		return list, fmt.Errorf("read URL list: %w", err)
	}

	return list, nil
}

// urlListAllocator 按列表顺序向多个 Worker 分配 URL。
type urlListAllocator struct {
	mu   sync.Mutex
	urls []string
	loop bool
	next int
}

// newURLListAllocator 创建 URL 分配器。
func newURLListAllocator(urls []string, loop bool) (*urlListAllocator, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("cannot create URL allocator with an empty list")
	}

	// 复制切片，避免调用者随后修改原始列表影响分配过程。
	copied := append([]string(nil), urls...)

	return &urlListAllocator{
		urls: copied,
		loop: loop,
	}, nil
}

// Next 返回下一个 URL。
// 单轮模式耗尽后返回 ("", false)；循环模式持续从头分配。
func (a *urlListAllocator) Next() (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.next >= len(a.urls) {
		if !a.loop {
			return "", false
		}
		a.next = 0
	}

	target := a.urls[a.next]
	a.next++
	return target, true
}
