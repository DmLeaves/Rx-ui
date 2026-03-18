package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"rxui/internal/model"
)

type TrafficStats struct {
	Tag      string `json:"tag"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

var statLineRe = regexp.MustCompile(`name:\s*"([^"]+)"\s+value:\s*([0-9]+)`) // xray api statsquery output

func queryXrayStats(pattern string) (string, error) {
	xrayBin := getXrayBinPath()
	cmd := exec.Command(xrayBin, "api", "statsquery", "--server=127.0.0.1:10085", "-pattern", pattern)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("xray api statsquery failed: %v, output: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func parseStatOutput(output string) map[string]int64 {
	result := map[string]int64{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := statLineRe.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		v, _ := strconv.ParseInt(m[2], 10, 64)
		result[m[1]] = v
	}
	return result
}

// getXrayStats 获取 Xray 入站流量统计（通过 Xray API）
func getXrayStats() ([]TrafficStats, error) {
	if !xrayRunning {
		return []TrafficStats{}, nil
	}

	out, err := queryXrayStats("inbound>>>")
	if err != nil {
		return nil, err
	}
	values := parseStatOutput(out)
	statsMap := map[string]*TrafficStats{}

	for name, v := range values {
		// inbound>>>tag>>>traffic>>>uplink/downlink
		parts := strings.Split(name, ">>>")
		if len(parts) < 4 || parts[0] != "inbound" {
			continue
		}
		tag := parts[1]
		dir := parts[3]
		if tag == "api" {
			continue
		}
		if _, ok := statsMap[tag]; !ok {
			statsMap[tag] = &TrafficStats{Tag: tag}
		}
		if dir == "uplink" {
			statsMap[tag].Uplink = v
		} else if dir == "downlink" {
			statsMap[tag].Downlink = v
		}
	}

	res := make([]TrafficStats, 0, len(statsMap))
	for _, s := range statsMap {
		res = append(res, *s)
	}
	return res, nil
}

// getInboundTraffic 获取单个入站的流量
func getInboundTraffic(tag string) (uplink, downlink int64, err error) {
	stats, err := getXrayStats()
	if err != nil {
		return 0, 0, err
	}
	for _, s := range stats {
		if s.Tag == tag {
			return s.Uplink, s.Downlink, nil
		}
	}
	return 0, 0, nil
}

// syncTrafficToDatabase 同步流量到数据库（写入绝对值）
func syncTrafficToDatabase() error {
	stats, err := getXrayStats()
	if err != nil {
		return err
	}
	for _, s := range stats {
		db.Model(&model.Inbound{}).Where("tag = ?", s.Tag).Updates(map[string]interface{}{
			"up":   s.Uplink,
			"down": s.Downlink,
		})
	}

	// 客户端流量（需要客户端在 settings 里带 email=clt-<id>）
	out, err := queryXrayStats("user>>>")
	if err == nil {
		values := parseStatOutput(out)
		for name, v := range values {
			// user>>>clt-123>>>traffic>>>uplink/downlink
			parts := strings.Split(name, ">>>")
			if len(parts) < 4 || parts[0] != "user" {
				continue
			}
			email := parts[1]
			dir := parts[3]
			if !strings.HasPrefix(email, "clt-") {
				continue
			}
			id, convErr := strconv.Atoi(strings.TrimPrefix(email, "clt-"))
			if convErr != nil {
				continue
			}
			if dir == "uplink" {
				db.Model(&model.Client{}).Where("id = ?", id).Update("up", v)
			} else if dir == "downlink" {
				db.Model(&model.Client{}).Where("id = ?", id).Update("down", v)
			}
		}
	}

	return nil
}

// startTrafficSyncJob 启动流量同步定时任务
func startTrafficSyncJob() {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			if xrayRunning {
				_ = syncTrafficToDatabase()
			}
		}
	}()
}

// formatTraffic 格式化流量显示
func formatTraffic(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
