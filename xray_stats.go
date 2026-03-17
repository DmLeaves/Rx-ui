package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

import (
	"rxui/internal/model"
)

type TrafficStats struct {
	Tag      string `json:"tag"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// getXrayStats 获取 Xray 流量统计（通过文件系统）
func getXrayStats() ([]TrafficStats, error) {
	if !xrayRunning {
		return nil, fmt.Errorf("Xray 未运行")
	}

	// Xray 将统计信息写入文件
	statsDir := "./data/stats"
	if _, err := os.Stat(statsDir); os.IsNotExist(err) {
		// 如果目录不存在，返回空
		return []TrafficStats{}, nil
	}

	statsMap := make(map[string]*TrafficStats)

	// 读取所有统计文件
	err := filepath.Walk(statsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 解析文件名格式：inbound_tag_uplink 或 inbound_tag_downlink
		filename := info.Name()
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			return nil
		}

		if parts[0] != "inbound" {
			return nil
		}

		tag := parts[1]
		direction := parts[2]

		// 读取文件内容（包含流量值）
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		value, err := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64)
		if err != nil {
			return nil
		}

		if _, ok := statsMap[tag]; !ok {
			statsMap[tag] = &TrafficStats{Tag: tag}
		}

		if direction == "uplink" {
			statsMap[tag].Uplink = value
		} else if direction == "downlink" {
			statsMap[tag].Downlink = value
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("读取统计文件失败: %v", err)
	}

	// 转换为切片
	result := make([]TrafficStats, 0, len(statsMap))
	for _, s := range statsMap {
		// 跳过 api 入站
		if s.Tag != "api" {
			result = append(result, *s)
		}
	}

	return result, nil
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

// syncTrafficToDatabase 同步流量到数据库
func syncTrafficToDatabase() error {
	stats, err := getXrayStats()
	if err != nil {
		return err
	}

	for _, s := range stats {
		// 查找对应的入站规则
		var inbound model.Inbound
		if err := db.Where("tag = ?", s.Tag).First(&inbound).Error; err != nil {
			continue
		}

		// 更新流量
		inbound.Up += s.Uplink
		inbound.Down += s.Downlink
		db.Save(&inbound)
	}

	return nil
}

// resetXrayStats 重置 Xray 流量统计（在同步后调用）
func resetXrayStats() error {
	// 删除统计文件
	statsDir := "./data/stats"
	if _, err := os.Stat(statsDir); os.IsNotExist(err) {
		return nil
	}

	// 清空文件内容
	err := filepath.Walk(statsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 清空文件内容
		return os.WriteFile(path, []byte("0"), 0644)
	})

	return err
}

// startTrafficSyncJob 启动流量同步定时任务
func startTrafficSyncJob() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			if xrayRunning {
				if err := syncTrafficToDatabase(); err == nil {
					resetXrayStats()
				}
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

// 读取流量文件（辅助函数）
func readTrafficFile(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		value, err := strconv.ParseInt(scanner.Text(), 10, 64)
		if err != nil {
			return 0, err
		}
		return value, nil
	}

	return 0, fmt.Errorf("文件为空")
}
