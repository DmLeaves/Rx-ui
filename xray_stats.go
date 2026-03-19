package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"rxui/internal/model"
	"gorm.io/gorm"
)

type TrafficStats struct {
	Tag      string `json:"tag"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

var (
	statLineRe  = regexp.MustCompile(`name:\s*"([^"]+)"\s+value:\s*([0-9]+)`) // one-line fallback
	nameOnlyRe  = regexp.MustCompile(`name:\s*"([^"]+)"`)
	valueOnlyRe = regexp.MustCompile(`value:\s*([0-9]+)`)
)

var (
	trafficMu           sync.Mutex
	lastInboundUplink   = map[string]int64{}
	lastInboundDownlink = map[string]int64{}
	lastClientUplink    = map[int]int64{}
	lastClientDownlink  = map[int]int64{}
)

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

	// 优先解析 xray api statsquery 的 JSON 输出
	type statItem struct {
		Name  string `json:"name"`
		Value int64  `json:"value"`
	}
	type statResp struct {
		Stat []statItem `json:"stat"`
	}
	var sr statResp
	if err := json.Unmarshal([]byte(output), &sr); err == nil && len(sr.Stat) > 0 {
		for _, s := range sr.Stat {
			if strings.TrimSpace(s.Name) == "" {
				continue
			}
			result[s.Name] = s.Value
		}
		return result
	}

	// 回退：兼容文本输出（单行/多行）
	var pendingName string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := statLineRe.FindStringSubmatch(line); len(m) == 3 {
			v, _ := strconv.ParseInt(m[2], 10, 64)
			result[m[1]] = v
			pendingName = ""
			continue
		}
		if m := nameOnlyRe.FindStringSubmatch(line); len(m) == 2 {
			pendingName = m[1]
			continue
		}
		if pendingName != "" {
			if m := valueOnlyRe.FindStringSubmatch(line); len(m) == 2 {
				v, _ := strconv.ParseInt(m[1], 10, 64)
				result[pendingName] = v
				pendingName = ""
			}
		}
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

// syncTrafficToDatabase 同步流量到数据库（累计增量）
func syncTrafficToDatabase() error {
	stats, err := getXrayStats()
	if err != nil {
		return err
	}

	trafficMu.Lock()
	defer trafficMu.Unlock()

	for _, s := range stats {
		prevUp, hasPrevUp := lastInboundUplink[s.Tag]
		prevDown, hasPrevDown := lastInboundDownlink[s.Tag]

		deltaUp := int64(0)
		deltaDown := int64(0)
		if hasPrevUp {
			deltaUp = s.Uplink - prevUp
			if deltaUp < 0 {
				// Xray 重启后计数器归零，当前值作为新的增量
				deltaUp = s.Uplink
			}
		}
		if hasPrevDown {
			deltaDown = s.Downlink - prevDown
			if deltaDown < 0 {
				deltaDown = s.Downlink
			}
		}

		lastInboundUplink[s.Tag] = s.Uplink
		lastInboundDownlink[s.Tag] = s.Downlink

		if hasPrevUp || hasPrevDown {
			if deltaUp > 0 || deltaDown > 0 {
				db.Model(&model.Inbound{}).Where("tag = ?", s.Tag).Updates(map[string]interface{}{
					"up":       gorm.Expr("up + ?", deltaUp),
					"down":     gorm.Expr("down + ?", deltaDown),
					"all_time": gorm.Expr("COALESCE(all_time, 0) + ?", deltaUp+deltaDown),
				})
			}
		}
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
				prev, hasPrev := lastClientUplink[id]
				delta := int64(0)
				if hasPrev {
					delta = v - prev
					if delta < 0 {
						delta = v
					}
				}
				lastClientUplink[id] = v
				if hasPrev && delta > 0 {
					db.Model(&model.Client{}).Where("id = ?", id).Update("up", gorm.Expr("up + ?", delta))
				}
			} else if dir == "downlink" {
				prev, hasPrev := lastClientDownlink[id]
				delta := int64(0)
				if hasPrev {
					delta = v - prev
					if delta < 0 {
						delta = v
					}
				}
				lastClientDownlink[id] = v
				if hasPrev && delta > 0 {
					db.Model(&model.Client{}).Where("id = ?", id).Update("down", gorm.Expr("down + ?", delta))
				}
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
