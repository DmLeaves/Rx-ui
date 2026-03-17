package handler

import (
	"runtime"
	"time"

	"rxui/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

// SystemHandler 系统信息处理器
type SystemHandler struct {
	xrayService *service.XrayService
	startTime   time.Time
}

// NewSystemHandler 创建系统信息处理器
func NewSystemHandler(xrayService *service.XrayService) *SystemHandler {
	return &SystemHandler{
		xrayService: xrayService,
		startTime:   time.Now(),
	}
}

// RegisterRoutes 注册路由
func (h *SystemHandler) RegisterRoutes(r *gin.RouterGroup) {
	system := r.Group("/system")
	{
		system.GET("/status", h.GetStatus)
		system.POST("/xray/restart", h.RestartXray)
		system.GET("/xray/version", h.GetXrayVersion)
	}
}

// SystemStatus 系统状态
type SystemStatus struct {
	// 系统信息
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Uptime   int64  `json:"uptime"` // 系统运行时间（秒）

	// CPU
	CPUPercent float64 `json:"cpuPercent"`
	CPUCores   int     `json:"cpuCores"`

	// 内存
	MemTotal   uint64  `json:"memTotal"`   // 总内存（字节）
	MemUsed    uint64  `json:"memUsed"`    // 已用内存
	MemPercent float64 `json:"memPercent"` // 使用率

	// 磁盘
	DiskTotal   uint64  `json:"diskTotal"`
	DiskUsed    uint64  `json:"diskUsed"`
	DiskPercent float64 `json:"diskPercent"`

	// 网络
	NetUpload   uint64 `json:"netUpload"`   // 上传字节
	NetDownload uint64 `json:"netDownload"` // 下载字节

	// Xray 状态
	XrayRunning bool   `json:"xrayRunning"`
	XrayVersion string `json:"xrayVersion"`

	// 面板信息
	PanelUptime int64  `json:"panelUptime"` // 面板运行时间（秒）
	GoVersion   string `json:"goVersion"`
}

// GetStatus 获取系统状态
// @Summary 获取系统状态
// @Tags System
// @Produce json
// @Success 200 {object} Response{data=SystemStatus}
// @Router /api/v1/system/status [get]
func (h *SystemHandler) GetStatus(c *gin.Context) {
	status := SystemStatus{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CPUCores:    runtime.NumCPU(),
		GoVersion:   runtime.Version(),
		PanelUptime: int64(time.Since(h.startTime).Seconds()),
		XrayRunning: h.xrayService.IsRunning(),
	}

	// 主机信息
	if hostInfo, err := host.Info(); err == nil {
		status.Hostname = hostInfo.Hostname
		status.Uptime = int64(hostInfo.Uptime)
	}

	// CPU 使用率
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		status.CPUPercent = cpuPercent[0]
	}

	// 内存信息
	if memInfo, err := mem.VirtualMemory(); err == nil {
		status.MemTotal = memInfo.Total
		status.MemUsed = memInfo.Used
		status.MemPercent = memInfo.UsedPercent
	}

	// 磁盘信息
	if diskInfo, err := disk.Usage("/"); err == nil {
		status.DiskTotal = diskInfo.Total
		status.DiskUsed = diskInfo.Used
		status.DiskPercent = diskInfo.UsedPercent
	}

	// 网络信息
	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		status.NetUpload = netIO[0].BytesSent
		status.NetDownload = netIO[0].BytesRecv
	}

	Success(c, status)
}

// RestartXray 重启 Xray
// @Summary 重启 Xray
// @Tags System
// @Produce json
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/system/xray/restart [post]
func (h *SystemHandler) RestartXray(c *gin.Context) {
	if err := h.xrayService.Restart(); err != nil {
		ServerError(c, "重启 Xray 失败: "+err.Error())
		return
	}
	SuccessMsg(c, "Xray 已重启", nil)
}

// XrayVersionResponse Xray 版本响应
type XrayVersionResponse struct {
	Version string `json:"version"`
	Running bool   `json:"running"`
}

// GetXrayVersion 获取 Xray 版本
// @Summary 获取 Xray 版本
// @Tags System
// @Produce json
// @Success 200 {object} Response{data=XrayVersionResponse}
// @Router /api/v1/system/xray/version [get]
func (h *SystemHandler) GetXrayVersion(c *gin.Context) {
	// TODO: 执行 xray -version 获取版本
	Success(c, XrayVersionResponse{
		Version: "1.8.0", // TODO: 动态获取
		Running: h.xrayService.IsRunning(),
	})
}
