package main

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"rxui/internal/model"
	"rxui/internal/web"
)

var (
	db          *gorm.DB
	xrayProcess *os.Process
	xrayRunning bool
	startTime   = time.Now()
)

func initDatabase() {
	// 确保数据目录存在
	os.MkdirAll("./data", 0755)

	var err error
	db, err = gorm.Open(sqlite.Open("./data/rx-ui.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 自动迁移
	db.AutoMigrate(&model.User{}, &model.Inbound{}, &model.Client{}, &model.Certificate{}, &model.FirewallRule{}, &model.Setting{})

	// 创建默认用户（如果不存在）
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		defaultUser := &model.User{Username: "admin", Password: "admin123", Enable: true}
		db.Create(defaultUser)
		log.Println("Created default user: admin / admin123")
	}

	loadSettingsFromDB()
}

func handleSettingCLI(args []string) {
	fs := flag.NewFlagSet("setting", flag.ExitOnError)
	show := fs.Bool("show", false, "show current settings")
	port := fs.String("port", "", "set panel port")
	username := fs.String("username", "", "set admin username")
	password := fs.String("password", "", "set admin password")
	reset := fs.Bool("reset", false, "reset panel settings to defaults")
	_ = fs.Parse(args)

	if *reset {
		for k, v := range defaultSettings {
			settings[k] = v
			upsertSetting(k, v)
		}
		fmt.Println("面板设置已重置为默认值")
	}

	if strings.TrimSpace(*port) != "" {
		settings["webPort"] = strings.TrimSpace(*port)
		upsertSetting("webPort", settings["webPort"])
		fmt.Printf("面板端口已设置为: %s\n", settings["webPort"])
	}

	if strings.TrimSpace(*username) != "" || strings.TrimSpace(*password) != "" {
		var user model.User
		if err := db.Order("id asc").First(&user).Error; err != nil {
			fmt.Printf("更新账户失败: %v\n", err)
			return
		}
		if strings.TrimSpace(*username) != "" {
			user.Username = strings.TrimSpace(*username)
		}
		if strings.TrimSpace(*password) != "" {
			user.Password = strings.TrimSpace(*password)
		}
		db.Save(&user)
		fmt.Println("管理员账号信息已更新")
	}

	if *show || (*port == "" && *username == "" && *password == "" && !*reset) {
		fmt.Printf("webPort: %s\n", settings["webPort"])
		fmt.Printf("webBasePath: %s\n", settings["webBasePath"])
		fmt.Printf("timeZone: %s\n", settings["timeZone"])
		var user model.User
		if err := db.Order("id asc").First(&user).Error; err == nil {
			fmt.Printf("username: %s\n", user.Username)
		}
	}
}

func main() {
	initDatabase()

	if len(os.Args) > 1 && os.Args[1] == "setting" {
		handleSettingCLI(os.Args[2:])
		return
	}

	// 自动安装 Xray
	if err := ensureXrayInstalled(); err != nil {
		log.Printf("警告: Xray 安装失败: %v", err)
	}

	// 启动面板时自动启动 Xray（安装后/重启后无需手动启动）
	if err := startXray(); err != nil {
		log.Printf("警告: Xray 自动启动失败: %v", err)
	} else {
		log.Printf("Xray 已自动启动")
	}

	// 设置 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 设置 API 路由
	api := r.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", handleHealth)

		// 认证 API
		auth := api.Group("/auth")
		{
			auth.POST("/login", handleLogin)
			auth.GET("/me", handleMe)
		}

		// 入站规则 API
		api.GET("/inbounds", handleGetInbounds)
		api.POST("/inbounds", handleCreateInbound)
		api.PUT("/inbounds/:id", handleUpdateInbound)
		api.DELETE("/inbounds/:id", handleDeleteInbound)
		api.POST("/inbounds/:id/resetTraffic", handleResetInboundTraffic)

		// 客户端 API（使用 /clients 独立路由）
		api.GET("/clients", handleGetClients)          // ?inboundId=xxx
		api.POST("/clients", handleCreateClient)       // body 包含 inboundId
		api.PUT("/clients/:id", handleUpdateClient)
		api.DELETE("/clients/:id", handleDeleteClient)

		// 用户管理 API
		users := api.Group("/users")
		{
			users.GET("", handleGetUsers)
			users.POST("", handleCreateUser)
			users.PUT("/:id/password", handleChangePassword)
			users.DELETE("/:id", handleDeleteUser)
		}

		// 证书管理 API
		certs := api.Group("/certificates")
		{
			certs.GET("", handleGetCertificates)
			certs.GET("/expiring", handleGetExpiringCertificates)
			certs.POST("", handleCreateCertificate)
			certs.PUT("/:id", handleUpdateCertificate)
			certs.GET("/acme/status", handleGetAcmeStatus)
			certs.POST("/:id/reload", handleReloadCertificate)
			certs.POST("/:id/renew", handleRenewCertificate)
			certs.DELETE("/:id", handleDeleteCertificate)
		}

		// 系统设置 API
		api.GET("/settings", handleGetSettings)
		api.PUT("/settings", handleUpdateSettings)

		// 防火墙管理 API
		firewall := api.Group("/firewall")
		{
			firewall.GET("/rules", handleListFirewallRules)
			firewall.POST("/rules", handleCreateFirewallRule)
			firewall.PUT("/rules/:id", handleUpdateFirewallRule)
			firewall.DELETE("/rules/:id", handleDeleteFirewallRule)
			firewall.POST("/reconcile", handleReconcileFirewall)
		}

		// Xray 控制 API
		xray := api.Group("/xray")
		{
			xray.GET("/status", handleXrayStatus)
			xray.POST("/start", handleXrayStart)
			xray.POST("/stop", handleXrayStop)
			xray.POST("/restart", handleXrayRestart)
		}

		// 流量统计 API
		api.GET("/traffic", handleGetTraffic)
		api.GET("/traffic/:tag", handleGetTrafficByTag)

		// 系统信息 API
		api.GET("/system/status", handleSystemStatus)

		// 订阅 API
		api.GET("/sub", handleSubscription)
	}

	// 设置静态文件服务
	if err := web.SetupStaticFiles(r); err != nil {
		log.Printf("Warning: Failed to setup static files: %v", err)
	}

	// 启动后台任务
	startTrafficSyncJob()
	startCertRenewJob()
	go reconcileFirewall()

	// 启动服务器（优先级：PORT 环境变量 > settings.webPort > 默认 54321）
	port := strings.TrimSpace(settings["webPort"])
	if port == "" {
		port = "54321"
	}
	if p := os.Getenv("PORT"); strings.TrimSpace(p) != "" {
		port = strings.TrimSpace(p)
	}

	go func() {
		log.Printf("Rx-ui starting on :%s", port)
		if err := r.Run(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	stopXray()
	time.Sleep(1 * time.Second)
	log.Println("Goodbye!")
}

// ===== 健康检查 =====

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"status": "healthy"}})
}

// ===== 认证 API =====

func handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "请输入用户名和密码"})
		return
	}

	var user model.User
	if err := db.Where("username = ? AND password = ?", req.Username, req.Password).First(&user).Error; err != nil {
		c.JSON(401, gin.H{"code": 1, "message": "用户名或密码错误"})
		return
	}

	if !user.Enable {
		c.JSON(403, gin.H{"code": 1, "message": "账户已禁用"})
		return
	}

	token := "rx-ui-token-" + user.Username

	c.JSON(200, gin.H{
		"code":    0,
		"message": "登录成功",
		"data": gin.H{
			"token": token,
			"user":  gin.H{"id": user.ID, "username": user.Username},
		},
	})
}

func handleMe(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"id": 1, "username": "admin"}})
}

// ===== 入站规则 API =====

func handleGetInbounds(c *gin.Context) {
	var inbounds []model.Inbound
	db.Find(&inbounds)
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": inbounds})
}

func handleCreateInbound(c *gin.Context) {
	var inbound model.Inbound
	if err := c.ShouldBindJSON(&inbound); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if inbound.Tag == "" {
		inbound.Tag = fmt.Sprintf("inbound-%d", inbound.Port)
	}
	if err := db.Create(&inbound).Error; err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "创建失败: " + err.Error()})
		return
	}
	if err := applyInboundRuntimeChanges(); err != nil {
		_ = db.Delete(&model.Inbound{}, inbound.ID).Error
		c.JSON(500, gin.H{"code": 1, "message": "入站已回滚，Xray 应用失败: " + err.Error()})
		return
	}
	go reconcileFirewall()
	msg := "创建成功"
	if xrayRunning {
		msg = "创建成功，已应用到 Xray"
	}
	c.JSON(201, gin.H{"code": 0, "message": msg, "data": inbound})
}

func handleUpdateInbound(c *gin.Context) {
	id := c.Param("id")
	var oldInbound model.Inbound
	if err := db.First(&oldInbound, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "入站规则不存在"})
		return
	}
	inbound := oldInbound
	if err := c.ShouldBindJSON(&inbound); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}
	inbound.ID = oldInbound.ID
	if err := db.Save(&inbound).Error; err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "更新失败: " + err.Error()})
		return
	}
	if err := applyInboundRuntimeChanges(); err != nil {
		_ = db.Save(&oldInbound).Error
		c.JSON(500, gin.H{"code": 1, "message": "更新已回滚，Xray 应用失败: " + err.Error()})
		return
	}
	go reconcileFirewall()
	msg := "更新成功"
	if xrayRunning {
		msg = "更新成功，已应用到 Xray"
	}
	c.JSON(200, gin.H{"code": 0, "message": msg, "data": inbound})
}

func applyInboundRuntimeChanges() error {
	if !xrayRunning {
		return nil
	}
	if err := generateXrayConfig(); err != nil {
		return fmt.Errorf("生成配置失败: %w", err)
	}
	xrayBin := getXrayBinPath()
	testCmd := exec.Command(xrayBin, "run", "-test", "-c", "./data/xray.json")
	if out, err := testCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("配置校验失败: %v, 输出: %s", err, strings.TrimSpace(string(out)))
	}
	if err := stopXray(); err != nil {
		return fmt.Errorf("停止 Xray 失败: %w", err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := startXray(); err != nil {
		return fmt.Errorf("启动 Xray 失败: %w", err)
	}
	return nil
}

func handleDeleteInbound(c *gin.Context) {
	id := c.Param("id")
	// 删除关联的客户端
	db.Where("inbound_id = ?", id).Delete(&model.Client{})
	db.Delete(&model.Inbound{}, id)
	go reconcileFirewall()
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func handleResetInboundTraffic(c *gin.Context) {
	id := c.Param("id")
	db.Model(&model.Inbound{}).Where("id = ?", id).Updates(map[string]interface{}{"up": 0, "down": 0})
	c.JSON(200, gin.H{"code": 0, "message": "流量已重置"})
}

// ===== 客户端 API =====

func handleGetClients(c *gin.Context) {
	inboundId := c.Query("inboundId")
	var clients []model.Client
	if inboundId != "" {
		db.Where("inbound_id = ?", inboundId).Find(&clients)
	} else {
		db.Find(&clients)
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": clients})
}

func syncInboundClientsToSettings(inboundID int) error {
	var inbound model.Inbound
	if err := db.First(&inbound, inboundID).Error; err != nil {
		return err
	}

	var settings map[string]interface{}
	if strings.TrimSpace(inbound.Settings) != "" {
		_ = json.Unmarshal([]byte(inbound.Settings), &settings)
	}
	if settings == nil {
		settings = map[string]interface{}{}
	}

	var clients []model.Client
	db.Where("inbound_id = ? AND enable = ?", inboundID, true).Order("id asc").Find(&clients)

	switch inbound.Protocol {
	case model.ProtocolVMess:
		arr := make([]map[string]interface{}, 0)
		for _, c := range clients {
			if strings.TrimSpace(c.UUID) == "" {
				continue
			}
			arr = append(arr, map[string]interface{}{"id": c.UUID, "alterId": 0, "email": fmt.Sprintf("clt-%d", c.ID)})
		}
		settings["clients"] = arr
	case model.ProtocolVLESS:
		arr := make([]map[string]interface{}, 0)
		for _, c := range clients {
			if strings.TrimSpace(c.UUID) == "" {
				continue
			}
			arr = append(arr, map[string]interface{}{"id": c.UUID, "flow": c.Flow, "email": fmt.Sprintf("clt-%d", c.ID)})
		}
		settings["clients"] = arr
		if _, ok := settings["decryption"]; !ok {
			settings["decryption"] = "none"
		}
	case model.ProtocolTrojan:
		arr := make([]map[string]interface{}, 0)
		for _, c := range clients {
			if strings.TrimSpace(c.Password) == "" {
				continue
			}
			arr = append(arr, map[string]interface{}{"password": c.Password, "email": fmt.Sprintf("clt-%d", c.ID)})
		}
		settings["clients"] = arr
	case model.ProtocolShadowsocks:
		for _, c := range clients {
			if strings.TrimSpace(c.Password) != "" {
				settings["password"] = c.Password
				break
			}
		}
	}

	b, _ := json.Marshal(settings)
	inbound.Settings = string(b)
	if err := db.Save(&inbound).Error; err != nil {
		return err
	}

	if xrayRunning {
		_ = stopXray()
		time.Sleep(300 * time.Millisecond)
		_ = startXray()
	}
	return nil
}

func handleCreateClient(c *gin.Context) {
	var client model.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}
	if client.InboundID == 0 {
		c.JSON(400, gin.H{"code": 1, "message": "inboundId 不能为空"})
		return
	}
	if err := db.Create(&client).Error; err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "创建失败"})
		return
	}
	_ = syncInboundClientsToSettings(client.InboundID)
	c.JSON(201, gin.H{"code": 0, "message": "创建成功", "data": client})
}

func handleUpdateClient(c *gin.Context) {
	id := c.Param("id")
	var client model.Client
	if err := db.First(&client, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "客户端不存在"})
		return
	}
	inboundID := client.InboundID
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}
	client.InboundID = inboundID
	if err := db.Save(&client).Error; err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "更新失败"})
		return
	}
	_ = syncInboundClientsToSettings(client.InboundID)
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": client})
}

func handleDeleteClient(c *gin.Context) {
	id := c.Param("id")
	var client model.Client
	if err := db.First(&client, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "客户端不存在"})
		return
	}
	if err := db.Delete(&model.Client{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "删除失败"})
		return
	}
	_ = syncInboundClientsToSettings(client.InboundID)
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

// ===== 用户管理 API =====

func handleGetUsers(c *gin.Context) {
	var users []model.User
	db.Find(&users)
	result := make([]gin.H, len(users))
	for i, u := range users {
		result[i] = gin.H{"id": u.ID, "username": u.Username, "enable": u.Enable, "createdAt": u.CreatedAt}
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": result})
}

func handleCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "请输入用户名和密码（密码至少6位）"})
		return
	}
	var existing model.User
	if db.Where("username = ?", req.Username).First(&existing).Error == nil {
		c.JSON(400, gin.H{"code": 1, "message": "用户名已存在"})
		return
	}
	user := model.User{Username: req.Username, Password: req.Password, Enable: true}
	db.Create(&user)
	c.JSON(201, gin.H{"code": 0, "message": "用户已创建"})
}

func handleChangePassword(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "密码至少6位"})
		return
	}
	var user model.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "用户不存在"})
		return
	}
	user.Password = req.NewPassword
	db.Save(&user)
	c.JSON(200, gin.H{"code": 0, "message": "密码已更新"})
}

func handleDeleteUser(c *gin.Context) {
	id := c.Param("id")
	db.Delete(&model.User{}, id)
	c.JSON(200, gin.H{"code": 0, "message": "用户已删除"})
}

// ===== 证书管理 API =====

func handleGetCertificates(c *gin.Context) {
	var certs []model.Certificate
	db.Find(&certs)
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": certs})
}

type certificateUpsertRequest struct {
	Domain      string `json:"domain"`
	CertFile    string `json:"certFile"`
	KeyFile     string `json:"keyFile"`
	CertContent string `json:"certContent"`
	KeyContent  string `json:"keyContent"`
	Remark      string `json:"remark"`
	AutoRenew   bool   `json:"autoRenew"`
}

func sanitizeDomainForCertPath(domain string) string {
	d := strings.TrimSpace(domain)
	if d == "" {
		d = "cert"
	}
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return re.ReplaceAllString(d, "_")
}

func ensureCertificateFiles(cert *model.Certificate) error {
	if strings.TrimSpace(cert.CertFile) != "" && strings.TrimSpace(cert.KeyFile) != "" {
		return nil
	}
	if strings.TrimSpace(cert.CertContent) == "" || strings.TrimSpace(cert.KeyContent) == "" {
		return nil
	}

	certDir := filepath.Join("data", "certs")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		return err
	}
	base := fmt.Sprintf("%s-%d", sanitizeDomainForCertPath(cert.Domain), time.Now().Unix())
	certPath := filepath.Join(certDir, base+".crt")
	keyPath := filepath.Join(certDir, base+".key")
	if err := os.WriteFile(certPath, []byte(strings.TrimSpace(cert.CertContent)+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, []byte(strings.TrimSpace(cert.KeyContent)+"\n"), 0o600); err != nil {
		return err
	}
	cert.CertFile = certPath
	cert.KeyFile = keyPath
	return nil
}

func handleCreateCertificate(c *gin.Context) {
	var req certificateUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}

	if strings.TrimSpace(req.Domain) == "" {
		c.JSON(400, gin.H{"code": 1, "message": "domain 不能为空"})
		return
	}

	cert := model.Certificate{
		Domain:      strings.TrimSpace(req.Domain),
		CertFile:    strings.TrimSpace(req.CertFile),
		KeyFile:     strings.TrimSpace(req.KeyFile),
		CertContent: req.CertContent,
		KeyContent:  req.KeyContent,
		Remark:      req.Remark,
		AutoRenew:   req.AutoRenew,
	}
	if err := ensureCertificateFiles(&cert); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "证书落盘失败: " + err.Error()})
		return
	}
	fillCertMeta(&cert)
	db.Create(&cert)
	c.JSON(201, gin.H{"code": 0, "message": "创建成功", "data": cert})
}

func handleUpdateCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert model.Certificate
	if err := db.First(&cert, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "证书不存在"})
		return
	}
	var req certificateUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}

	if strings.TrimSpace(req.Domain) != "" {
		cert.Domain = strings.TrimSpace(req.Domain)
	}
	if strings.TrimSpace(req.CertFile) != "" {
		cert.CertFile = strings.TrimSpace(req.CertFile)
	}
	if strings.TrimSpace(req.KeyFile) != "" {
		cert.KeyFile = strings.TrimSpace(req.KeyFile)
	}
	if strings.TrimSpace(req.CertContent) != "" {
		cert.CertContent = req.CertContent
	}
	if strings.TrimSpace(req.KeyContent) != "" {
		cert.KeyContent = req.KeyContent
	}
	cert.Remark = req.Remark
	cert.AutoRenew = req.AutoRenew

	if err := ensureCertificateFiles(&cert); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "证书落盘失败: " + err.Error()})
		return
	}
	fillCertMeta(&cert)
	db.Save(&cert)
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": cert})
}

func handleGetExpiringCertificates(c *gin.Context) {
	days := 30
	if q := c.Query("days"); q != "" {
		fmt.Sscanf(q, "%d", &days)
		if days <= 0 {
			days = 30
		}
	}

	deadline := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	var certs []model.Certificate
	db.Where("expires_at IS NOT NULL AND expires_at != ? AND expires_at <= ?", time.Time{}, deadline).Order("expires_at asc").Find(&certs)

	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": certs})
}

func handleReloadCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert model.Certificate
	if err := db.First(&cert, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "证书不存在"})
		return
	}
	fillCertMeta(&cert)
	db.Save(&cert)
	c.JSON(200, gin.H{"code": 0, "message": "证书信息已刷新", "data": cert})
}

func handleDeleteCertificate(c *gin.Context) {
	id := c.Param("id")
	db.Delete(&model.Certificate{}, id)
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func getAcmeStatus() (bool, []string) {
	missing := make([]string, 0)
	if strings.TrimSpace(settings["acmeEnabled"]) != "true" {
		missing = append(missing, "请先在系统设置启用 ACME")
	}
	if strings.TrimSpace(settings["acmeEmail"]) == "" {
		missing = append(missing, "acmeEmail")
	}
	provider := strings.TrimSpace(settings["acmeDnsProvider"])
	if provider == "" {
		missing = append(missing, "acmeDnsProvider")
	} else if provider == "cloudflare" {
		if strings.TrimSpace(settings["acmeDnsApiToken"]) == "" {
			missing = append(missing, "acmeDnsApiToken")
		}
	} else {
		missing = append(missing, "暂不支持的 DNS provider: "+provider)
	}
	return len(missing) == 0, missing
}

func handleGetAcmeStatus(c *gin.Context) {
	ok, missing := getAcmeStatus()
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"configured": ok,
		"provider":   settings["acmeDnsProvider"],
		"email":      settings["acmeEmail"],
		"missing":    missing,
	}})
}

func runLegoRenew(domain string) error {
	ok, missing := getAcmeStatus()
	if !ok {
		return fmt.Errorf("ACME 配置不完整: %s", strings.Join(missing, ", "))
	}
	email := strings.TrimSpace(settings["acmeEmail"])
	dnsProvider := strings.TrimSpace(settings["acmeDnsProvider"])
	dnsToken := strings.TrimSpace(settings["acmeDnsApiToken"])

	basePath := "./data/lego"
	_ = os.MkdirAll(basePath, 0o755)

	cmd := exec.Command("lego",
		"--accept-tos",
		"--email", email,
		"--dns", dnsProvider,
		"--path", basePath,
		"-d", domain,
		"renew",
		"--days", "30",
	)
	cmd.Env = append(os.Environ(), "CLOUDFLARE_DNS_API_TOKEN="+dnsToken)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lego renew 失败: %v, 输出: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func applyLegoCertFiles(cert *model.Certificate) error {
	basePath := "./data/lego/certificates"
	crt := filepath.Join(basePath, cert.Domain+".crt")
	key := filepath.Join(basePath, cert.Domain+".key")
	if _, err := os.Stat(crt); err != nil {
		return fmt.Errorf("未找到证书文件: %s", crt)
	}
	if _, err := os.Stat(key); err != nil {
		return fmt.Errorf("未找到私钥文件: %s", key)
	}

	if strings.TrimSpace(cert.CertFile) == "" {
		cert.CertFile = filepath.Join("data", "certs", cert.Domain+".crt")
	}
	if strings.TrimSpace(cert.KeyFile) == "" {
		cert.KeyFile = filepath.Join("data", "certs", cert.Domain+".key")
	}
	_ = os.MkdirAll(filepath.Dir(cert.CertFile), 0o755)
	_ = os.MkdirAll(filepath.Dir(cert.KeyFile), 0o755)

	crtBytes, err := os.ReadFile(crt)
	if err != nil {
		return err
	}
	keyBytes, err := os.ReadFile(key)
	if err != nil {
		return err
	}
	if err = os.WriteFile(cert.CertFile, crtBytes, 0o644); err != nil {
		return err
	}
	if err = os.WriteFile(cert.KeyFile, keyBytes, 0o600); err != nil {
		return err
	}
	cert.CertContent = string(crtBytes)
	cert.KeyContent = string(keyBytes)
	fillCertMeta(cert)
	return nil
}

func renewCertificate(cert *model.Certificate) error {
	if strings.TrimSpace(cert.Domain) == "" {
		return fmt.Errorf("证书域名为空")
	}
	if err := runLegoRenew(cert.Domain); err != nil {
		return err
	}
	if err := applyLegoCertFiles(cert); err != nil {
		return err
	}
	return db.Save(cert).Error
}

func handleRenewCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert model.Certificate
	if err := db.First(&cert, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "证书不存在"})
		return
	}
	if err := renewCertificate(&cert); err != nil {
		c.JSON(500, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "续签成功", "data": cert})
}

func startCertRenewJob() {
	ticker := time.NewTicker(12 * time.Hour)
	go func() {
		for range ticker.C {
			var certs []model.Certificate
			db.Where("auto_renew = ?", true).Find(&certs)
			for i := range certs {
				if certs[i].ExpiresAt.IsZero() {
					continue
				}
				if time.Until(certs[i].ExpiresAt) > 30*24*time.Hour {
					continue
				}
				if err := renewCertificate(&certs[i]); err != nil {
					log.Printf("[WARN] cert renew failed (%s): %v", certs[i].Domain, err)
				} else {
					log.Printf("[INF] cert renewed: %s", certs[i].Domain)
				}
			}
		}
	}()
}

func fillCertMeta(cert *model.Certificate) {
	parsed := parseCert(cert.CertContent)
	if parsed == nil && cert.CertFile != "" {
		if b, err := os.ReadFile(cert.CertFile); err == nil {
			parsed = parseCert(string(b))
		}
	}
	if parsed != nil {
		cert.ExpiresAt = parsed.NotAfter
		cert.Issuer = parsed.Issuer.CommonName
	}
}

func parseCert(pemText string) *x509.Certificate {
	if strings.TrimSpace(pemText) == "" {
		return nil
	}
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil
	}
	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	return crt
}

// ===== 系统设置 API =====

var defaultSettings = map[string]string{
	"webPort":          "54321",
	"webBasePath":      "/",
	"webCertFile":      "",
	"webKeyFile":       "",
	"xrayBinPath":      "", // 自动检测
	"timeZone":         "Asia/Shanghai",
	"acmeEmail":        "",
	"acmeDnsProvider":  "cloudflare",
	"acmeDnsApiToken":  "",
	"acmeEnabled":      "false",
}

var settings = map[string]string{}

func loadSettingsFromDB() {
	for k, v := range defaultSettings {
		settings[k] = v
	}

	var rows []model.Setting
	db.Find(&rows)
	for _, row := range rows {
		settings[row.Key] = row.Value
	}

	for k, v := range defaultSettings {
		if _, ok := settings[k]; !ok || settings[k] == "" {
			settings[k] = v
		}
		upsertSetting(k, settings[k])
	}
}

func upsertSetting(key, value string) {
	var s model.Setting
	db.Where("key = ?", key).Assign(model.Setting{Value: value}).FirstOrCreate(&s, model.Setting{Key: key})
}

// getXrayBinPath 获取 Xray 二进制路径
func getXrayBinPath() string {
	if settings["xrayBinPath"] != "" {
		return settings["xrayBinPath"]
	}
	return getXrayPath()
}

func handleGetSettings(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": settings})
}

func handleUpdateSettings(c *gin.Context) {
	var newSettings map[string]string
	if err := c.ShouldBindJSON(&newSettings); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}
	for k, v := range newSettings {
		settings[k] = v
		upsertSetting(k, v)
	}
	go reconcileFirewall()
	c.JSON(200, gin.H{"code": 0, "message": "设置已更新"})
}

// ===== 防火墙管理 API =====

func handleListFirewallRules(c *gin.Context) {
	var rules []model.FirewallRule
	db.Order("id desc").Find(&rules)
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": rules})
}

func handleCreateFirewallRule(c *gin.Context) {
	var req model.FirewallRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}
	req.Scope = model.FirewallScopeCustom
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	if req.Source == "" {
		req.Source = "any"
	}
	if req.Action == "" {
		req.Action = "allow"
	}
	req.Status = model.FirewallStatusPending
	db.Create(&req)
	applied, removed, err := reconcileFirewall()
	if err != nil {
		c.JSON(200, gin.H{"code": 0, "message": "已创建，防火墙同步存在告警", "data": gin.H{"rule": req, "applied": applied, "removed": removed, "error": err.Error()}})
		return
	}
	c.JSON(201, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func handleUpdateFirewallRule(c *gin.Context) {
	id := c.Param("id")
	var existing model.FirewallRule
	if err := db.First(&existing, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "规则不存在"})
		return
	}
	var req model.FirewallRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
		return
	}
	existing.Port = req.Port
	existing.Protocol = req.Protocol
	existing.Source = req.Source
	existing.Action = req.Action
	existing.Status = model.FirewallStatusPending
	db.Save(&existing)
	reconcileFirewall()
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": existing})
}

func handleDeleteFirewallRule(c *gin.Context) {
	id := c.Param("id")
	var existing model.FirewallRule
	if err := db.First(&existing, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 1, "message": "规则不存在"})
		return
	}
	db.Delete(&existing)
	reconcileFirewall()
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func handleReconcileFirewall(c *gin.Context) {
	applied, removed, err := reconcileFirewall()
	if err != nil {
		c.JSON(200, gin.H{"code": 0, "message": "同步完成（有告警）", "data": gin.H{"applied": applied, "removed": removed, "error": err.Error(), "provider": detectFirewallProvider()}})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "同步完成", "data": gin.H{"applied": applied, "removed": removed, "provider": detectFirewallProvider()}})
}

// ===== Xray 控制 API =====

func handleXrayStatus(c *gin.Context) {
	version := "未安装"
	xrayBin := getXrayBinPath()
	if _, err := os.Stat(xrayBin); err == nil {
		out, err := exec.Command(xrayBin, "version").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				version = strings.TrimSpace(lines[0])
			}
		}
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"running": xrayRunning,
			"version": version,
		},
	})
}

func handleXrayStart(c *gin.Context) {
	if xrayRunning {
		c.JSON(400, gin.H{"code": 1, "message": "Xray 已在运行"})
		return
	}

	if err := startXray(); err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "启动失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "Xray 已启动"})
}

func handleXrayStop(c *gin.Context) {
	if !xrayRunning {
		c.JSON(400, gin.H{"code": 1, "message": "Xray 未运行"})
		return
	}

	if err := stopXray(); err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "停止失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "Xray 已停止"})
}

func handleXrayRestart(c *gin.Context) {
	stopXray()
	time.Sleep(1 * time.Second)

	if err := startXray(); err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "重启失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "Xray 已重启"})
}

func startXray() error {
	// 确保 Xray 已安装
	if err := ensureXrayInstalled(); err != nil {
		return fmt.Errorf("Xray 未安装: %v", err)
	}

	// 生成配置
	if err := generateXrayConfig(); err != nil {
		return fmt.Errorf("生成配置失败: %v", err)
	}

	// 启动 Xray
	xrayBin := getXrayBinPath()
	cmd := exec.Command(xrayBin, "run", "-c", "./data/xray.json")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	xrayProcess = cmd.Process
	xrayRunning = true

	go func() {
		cmd.Wait()
		xrayRunning = false
		xrayProcess = nil
	}()

	return nil
}

func stopXray() error {
	if xrayProcess != nil {
		if err := xrayProcess.Kill(); err != nil {
			return err
		}
		xrayProcess = nil
	}
	xrayRunning = false
	return nil
}

func generateXrayConfig() error {
	var inbounds []model.Inbound
	db.Where("enable = ?", true).Find(&inbounds)

	inboundConfigs := make([]map[string]interface{}, 0)
	for _, inbound := range inbounds {
		cfg := buildInboundConfig(&inbound)
		if cfg != nil {
			inboundConfigs = append(inboundConfigs, cfg)
		}
	}

	apiInbound := map[string]interface{}{
		"tag":      "api",
		"listen":   "127.0.0.1",
		"port":     10085,
		"protocol": "dokodemo-door",
		"settings": map[string]interface{}{
			"address": "127.0.0.1",
		},
	}
	inboundConfigs = append(inboundConfigs, apiInbound)

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"api": map[string]interface{}{
			"tag":      "api",
			"services": []string{"StatsService"},
		},
		"stats": map[string]interface{}{},
		"policy": map[string]interface{}{
			"levels": map[string]interface{}{
				"0": map[string]interface{}{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]interface{}{
				"statsInboundUplink":   true,
				"statsInboundDownlink": true,
			},
		},
		"inbounds": inboundConfigs,
		"outbounds": []map[string]interface{}{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "blocked"},
			{"protocol": "freedom", "tag": "api"},
		},
		"routing": map[string]interface{}{
			"rules": []map[string]interface{}{
				{"type": "field", "inboundTag": []string{"api"}, "outboundTag": "api"},
				{"type": "field", "outboundTag": "blocked", "ip": []string{"geoip:private"}},
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("./data/xray.json", data, 0644)
}

func buildInboundConfig(inbound *model.Inbound) map[string]interface{} {
	var settings map[string]interface{}
	json.Unmarshal([]byte(inbound.Settings), &settings)

	var streamSettings map[string]interface{}
	if inbound.StreamSettings != "" {
		json.Unmarshal([]byte(inbound.StreamSettings), &streamSettings)
	}

	var sniffing map[string]interface{}
	if inbound.Sniffing != "" {
		json.Unmarshal([]byte(inbound.Sniffing), &sniffing)
	}

	cfg := map[string]interface{}{
		"tag":      inbound.Tag,
		"port":     inbound.Port,
		"protocol": inbound.Protocol,
		"settings": settings,
	}

	if inbound.Listen != "" {
		cfg["listen"] = inbound.Listen
	}

	if streamSettings != nil {
		cfg["streamSettings"] = streamSettings
	}

	if sniffing != nil {
		cfg["sniffing"] = sniffing
	}

	return cfg
}

// ===== 系统状态 API =====

func handleSystemStatus(c *gin.Context) {
	// CPU 信息
	cpuPercent, _ := cpu.Percent(time.Second, false)
	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	// 内存信息
	memInfo, _ := mem.VirtualMemory()
	memTotal := uint64(0)
	memUsed := uint64(0)
	if memInfo != nil {
		memTotal = memInfo.Total
		memUsed = memInfo.Used
	}

	// 负载信息
	loadInfo, _ := load.Avg()
	loadAvg := []float64{0, 0, 0}
	if loadInfo != nil {
		loadAvg = []float64{loadInfo.Load1, loadInfo.Load5, loadInfo.Load15}
	}

	// 主机信息
	hostInfo, _ := host.Info()
	uptime := uint64(0)
	if hostInfo != nil {
		uptime = hostInfo.Uptime
	}

	// 入站规则流量统计
	var inbounds []model.Inbound
	db.Find(&inbounds)
	totalUp := int64(0)
	totalDown := int64(0)
	for _, i := range inbounds {
		totalUp += i.Up
		totalDown += i.Down
	}

	// Xray 版本
	xrayVersionStr := "未安装"
	xrayBin := getXrayBinPath()
	if _, err := os.Stat(xrayBin); err == nil {
		out, err := exec.Command(xrayBin, "version").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				xrayVersionStr = strings.TrimSpace(lines[0])
			}
		}
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"cpu": gin.H{
				"cores":   runtime.NumCPU(),
				"percent": cpuUsage,
			},
			"memory": gin.H{
				"total": memTotal,
				"used":  memUsed,
			},
			"load":   loadAvg,
			"uptime": uptime,
			"traffic": gin.H{
				"up":   totalUp,
				"down": totalDown,
			},
			"xray": gin.H{
				"running": xrayRunning,
				"version": xrayVersionStr,
			},
			"panelUptime": int64(time.Since(startTime).Seconds()),
			"inboundCount": len(inbounds),
		},
	})
}

// ===== 流量统计 API =====

func handleGetTraffic(c *gin.Context) {
	stats, err := getXrayStats()
	if err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "获取流量统计失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": stats})
}

func handleGetTrafficByTag(c *gin.Context) {
	tag := c.Param("tag")
	uplink, downlink, err := getInboundTraffic(tag)
	if err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "获取流量失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"tag":      tag,
			"uplink":   uplink,
			"downlink": downlink,
		},
	})
}

// ===== 订阅 API =====

func handleSubscription(c *gin.Context) {
	host := c.Query("host")
	if host == "" {
		host = c.Request.Host
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
	}

	var inbounds []model.Inbound
	db.Where("enable = ?", true).Find(&inbounds)

	links := make([]string, 0)
	for _, inbound := range inbounds {
		link := generateInboundLink(&inbound, host)
		if link != "" {
			links = append(links, link)
		}
	}

	subContent := base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n")))
	c.Header("Content-Type", "text/plain")
	c.String(200, subContent)
}

func generateInboundLink(inbound *model.Inbound, host string) string {
	var settings map[string]interface{}
	json.Unmarshal([]byte(inbound.Settings), &settings)

	var streamSettings map[string]interface{}
	if inbound.StreamSettings != "" {
		json.Unmarshal([]byte(inbound.StreamSettings), &streamSettings)
	}

	network := "tcp"
	security := "none"
	wsPath := ""
	wsHost := ""
	grpcServiceName := ""
	sni := ""

	if streamSettings != nil {
		if n, ok := streamSettings["network"].(string); ok {
			network = n
		}
		if s, ok := streamSettings["security"].(string); ok {
			security = s
		}
		if ws, ok := streamSettings["wsSettings"].(map[string]interface{}); ok {
			if p, ok := ws["path"].(string); ok {
				wsPath = p
			}
			if h, ok := ws["headers"].(map[string]interface{}); ok {
				if hh, ok := h["Host"].(string); ok {
					wsHost = hh
				}
			}
		}
		if grpc, ok := streamSettings["grpcSettings"].(map[string]interface{}); ok {
			if sn, ok := grpc["serviceName"].(string); ok {
				grpcServiceName = sn
			}
		}
		if tls, ok := streamSettings["tlsSettings"].(map[string]interface{}); ok {
			if s, ok := tls["serverName"].(string); ok {
				sni = s
			}
		}
	}

	remark := inbound.Remark
	if remark == "" {
		remark = fmt.Sprintf("%d", inbound.Port)
	}

	switch inbound.Protocol {
	case "vmess":
		clients, ok := settings["clients"].([]interface{})
		if !ok || len(clients) == 0 {
			return ""
		}
		client := clients[0].(map[string]interface{})
		uuid := client["id"].(string)
		alterId := 0
		if aid, ok := client["alterId"].(float64); ok {
			alterId = int(aid)
		}

		obj := map[string]interface{}{
			"v":    "2",
			"ps":   remark,
			"add":  host,
			"port": inbound.Port,
			"id":   uuid,
			"aid":  alterId,
			"net":  network,
			"type": "none",
			"host": wsHost,
			"path": wsPath,
			"tls":  security,
		}
		if network == "grpc" {
			obj["path"] = grpcServiceName
		}
		data, _ := json.Marshal(obj)
		return "vmess://" + base64.StdEncoding.EncodeToString(data)

	case "vless":
		clients, ok := settings["clients"].([]interface{})
		if !ok || len(clients) == 0 {
			return ""
		}
		client := clients[0].(map[string]interface{})
		uuid := client["id"].(string)
		flow := ""
		if f, ok := client["flow"].(string); ok {
			flow = f
		}

		params := fmt.Sprintf("type=%s&security=%s", network, security)
		if network == "ws" && wsPath != "" {
			params += "&path=" + wsPath
		}
		if network == "grpc" && grpcServiceName != "" {
			params += "&serviceName=" + grpcServiceName
		}
		if security == "tls" && sni != "" {
			params += "&sni=" + sni
		}
		if flow != "" {
			params += "&flow=" + flow
		}

		return fmt.Sprintf("vless://%s@%s:%d?%s#%s", uuid, host, inbound.Port, params, remark)

	case "trojan":
		clients, ok := settings["clients"].([]interface{})
		if !ok || len(clients) == 0 {
			return ""
		}
		client := clients[0].(map[string]interface{})
		password := client["password"].(string)

		params := ""
		if sni != "" {
			params = "?sni=" + sni
		}

		return fmt.Sprintf("trojan://%s@%s:%d%s#%s", password, host, inbound.Port, params, remark)

	case "shadowsocks":
		method, _ := settings["method"].(string)
		password, _ := settings["password"].(string)
		userinfo := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", method, password)))
		return fmt.Sprintf("ss://%s@%s:%d#%s", userinfo, host, inbound.Port, remark)
	}

	return ""
}
