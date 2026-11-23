package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type ispCoverageStat struct {
	Names    int
	Segments int
}

type ispRawStat struct {
	Name     string
	Segments int
}

func analyzeISPFile(path string) (map[ISP]*ispCoverageStat, []ispRawStat, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	defer file.Close()

	stats := make(map[ISP]*ispCoverageStat)
	unknown := make([]ispRawStat, 0)

	totalNames := 0
	totalSegments := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "\t", 2)
		if len(fields) != 2 {
			return nil, nil, 0, 0, fmt.Errorf("无效数据行: %q", scanner.Text())
		}

		segments, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("无效网段数量: %q", fields[0])
		}

		name := strings.TrimSpace(fields[1])
		category := NormalizeISP(name)

		stat := stats[category]
		if stat == nil {
			stat = &ispCoverageStat{}
			stats[category] = stat
		}

		stat.Names++
		stat.Segments += segments

		totalNames++
		totalSegments += segments

		if category == ISPUnknown {
			unknown = append(unknown, ispRawStat{
				Name:     name,
				Segments: segments,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, 0, 0, err
	}

	sort.Slice(unknown, func(i, j int) bool {
		if unknown[i].Segments == unknown[j].Segments {
			return unknown[i].Name < unknown[j].Name
		}
		return unknown[i].Segments > unknown[j].Segments
	})

	return stats, unknown, totalNames, totalSegments, nil
}

func printISPCoverage(t *testing.T, title, path string) {
	t.Helper()

	stats, unknown, totalNames, totalSegments, err := analyzeISPFile(path)
	if err != nil {
		t.Fatalf("%s: %v", title, err)
	}

	categories := []struct {
		ISP  ISP
		Name string
	}{
		{ISPChinaTelecom, "中国电信"},
		{ISPChinaUnicom, "中国联通"},
		{ISPChinaMobile, "中国移动"},
		{ISPChinaBroadnet, "中国广电"},
		{ISPCERNET, "中国教育网"},
		{ISPOther, "其他已识别网络"},
		{ISPUnknown, "未知网络"},
	}

	fmt.Printf("\n========== %s ==========\n", title)
	fmt.Printf("ISP 原始名称总数：%d\n", totalNames)
	fmt.Printf("IP 段记录总数：%d\n", totalSegments)

	knownNames := 0
	knownSegments := 0

	fmt.Printf("\n%-20s %12s %15s\n", "分类", "ISP名称数", "IP段记录数")

	for _, category := range categories {
		stat := stats[category.ISP]
		if stat == nil {
			continue
		}

		fmt.Printf("%-20s %12d %15d\n",
			category.Name, stat.Names, stat.Segments)

		if category.ISP != ISPUnknown {
			knownNames += stat.Names
			knownSegments += stat.Segments
		}
	}

	nameRate := 0.0
	segmentRate := 0.0

	if totalNames > 0 {
		nameRate = float64(knownNames) * 100 / float64(totalNames)
	}
	if totalSegments > 0 {
		segmentRate = float64(knownSegments) * 100 / float64(totalSegments)
	}

	fmt.Printf("\n已识别 ISP 名称覆盖率：%.2f%% (%d/%d)\n",
		nameRate, knownNames, totalNames)
	fmt.Printf("已识别 IP 段覆盖率：%.2f%% (%d/%d)\n",
		segmentRate, knownSegments, totalSegments)

	fmt.Println("\n---------- UNKNOWN 原始 ISP（前 30 名） ----------")
	limit := len(unknown)
	if limit > 30 {
		limit = 30
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("%8d\t%s\n", unknown[i].Segments, unknown[i].Name)
	}
}

func TestGeoIPISPCoverage(t *testing.T) {
	testDataDir := filepath.Join("testdata", "ip2region")

	printISPCoverage(
		t,
		"中国 IPv4 ISP 归一化覆盖率",
		filepath.Join(testDataDir, "china_ipv4_isp.tsv"),
	)

	printISPCoverage(
		t,
		"中国 IPv6 ISP 归一化覆盖率",
		filepath.Join(testDataDir, "china_ipv6_isp.tsv"),
	)
}
