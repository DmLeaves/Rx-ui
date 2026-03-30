package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"rxui/internal/model"
)

type actionReq struct {
	RequestID string                 `json:"requestId"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
}

type controlClient struct {
	ClientID  string `json:"clientId"`
	PublicKey string `json:"publicKey"`
	Enabled   bool   `json:"enabled"`
	Remark    string `json:"remark"`
}

type generateClientReq struct {
	Remark string `json:"remark"`
}

var (
	nonceMu     sync.Mutex
	seenNonces        = map[string]int64{}
	nonceWindow int64 = 120

	requestMu   sync.Mutex
	requestSeen = map[string]struct {
		At   int64
		Resp gin.H
	}{}
	requestWindow int64 = 600
)

func parseControlClients() map[string]controlClient {
	res := map[string]controlClient{}
	raw := strings.TrimSpace(settings["controlClients"])
	if raw == "" {
		return res
	}
	_ = json.Unmarshal([]byte(raw), &res)
	return res
}

func saveControlClients(m map[string]controlClient) {
	b, _ := json.Marshal(m)
	settings["controlClients"] = string(b)
	upsertSetting("controlClients", settings["controlClients"])
}

func handleControlBootstrap(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"protocol": "rxui-control-v1",
		"auth": gin.H{
			"type":           "ed25519-signature",
			"headers":        []string{"X-Rxui-Client", "X-Rxui-Timestamp", "X-Rxui-Nonce", "X-Rxui-Signature"},
			"signText":       "METHOD\\nPATH\\nTIMESTAMP\\nNONCE\\nSHA256_HEX(BODY)",
			"windowSec":      nonceWindow,
			"idempotencySec": requestWindow,
		},
		"entrypoints": gin.H{
			"discovery": "/api/v1/control/discovery",
			"manifest":  "/api/v1/control/manifest",
			"errors":    "/api/v1/control/errors",
			"query":     "/api/v1/control/query",
			"exec":      "/api/v1/control/exec",
			"audit":     "/api/v1/control/audit",
		},
		"usage": []string{
			"先调用 /api/v1/control/discovery 获取可用入口与运行端口信息",
			"再调用 /api/v1/control/manifest 获取 action、参数格式与模式(query/exec)",
			"遇到错误码时查询 /api/v1/control/errors 解释与建议处理方式",
		},
		"examples": gin.H{
			"query": gin.H{"requestId": "req-1", "action": "xray.status", "params": gin.H{}},
			"exec":  gin.H{"requestId": "req-2", "action": "xray.restart", "params": gin.H{}},
		},
	}})
}

func handleControlDiscovery(c *gin.Context) {
	port := strings.TrimSpace(settings["webPort"])
	if port == "" {
		port = "54321"
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"service": gin.H{
			"name":    "rx-ui",
			"webPort": port,
		},
		"catalog": gin.H{
			"bootstrap": "/api/v1/control/bootstrap",
			"manifest":  "/api/v1/control/manifest",
			"errors":    "/api/v1/control/errors",
			"query":     "/api/v1/control/query",
			"exec":      "/api/v1/control/exec",
			"audit":     "/api/v1/control/audit",
			"health":    "/api/v1/health",
			"settings":  "/api/v1/settings",
			"system":    "/api/v1/system/status",
		},
	}})
}

func handleControlManifest(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"protocol":      "rxui-control-v1",
		"requestSchema": gin.H{"requestId": "string(optional)", "action": "string(required)", "params": "object(required)"},
		"actions":       controlActionSpecs(),
		"queryActions":  controlCapabilities()["query"],
		"execActions":   controlCapabilities()["exec"],
		"notes": []string{
			"manifest 仅描述当前版本服务端支持的动作，以运行中服务为准",
			"不要依赖仓库静态文档推断能力；每次会话优先先读 manifest",
		},
	}})
}

func handleControlErrors(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": controlErrorCodes()})
}

func handleControlGenerateClient(c *gin.Context) {
	var req generateClientReq
	_ = c.ShouldBindJSON(&req)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		c.JSON(500, gin.H{"code": 1, "message": "生成密钥失败: " + err.Error()})
		return
	}
	clientID := fmt.Sprintf("ai-%d", time.Now().Unix())
	cc := controlClient{
		ClientID:  clientID,
		PublicKey: base64.StdEncoding.EncodeToString(pub),
		Enabled:   true,
		Remark:    strings.TrimSpace(req.Remark),
	}
	if cc.Remark == "" {
		cc.Remark = "generated"
	}
	m := parseControlClients()
	m[clientID] = cc
	saveControlClients(m)

	// 构建完整 URL（协议 + 主机 + 端口 + 路径）
	host := c.Request.Host
	if host == "" {
		// 回退到配置的端口
		port := strings.TrimSpace(settings["webPort"])
		if port == "" {
			port = "54321"
		}
		host = "127.0.0.1:" + port
	}
	// 判断是否 HTTPS（如果请求是 HTTPS 或面板配置了证书）
	scheme := "http"
	if c.Request.TLS != nil || (settings["webCertFile"] != "" && settings["webKeyFile"] != "") {
		scheme = "https"
	}
	
	basePath := strings.TrimRight(strings.TrimSpace(settings["webBasePath"]), "/")
	if basePath == "" {
		basePath = ""
	}
	skillPath := basePath + "/api/v1/control/skill"
	if !strings.HasPrefix(skillPath, "/") {
		skillPath = "/" + skillPath
	}
	skillURL := fmt.Sprintf("%s://%s%s", scheme, host, skillPath)

	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"clientId":   clientID,
		"publicKey":  cc.PublicKey,
		"privateKey": base64.StdEncoding.EncodeToString(priv),
		"skillUrl":   skillURL,
		"hint":       "私钥只在此返回一次，请立即保存。",
	}})
}

func handleControlSkill(c *gin.Context) {
	path := "skills/rxui-control-discovery/SKILL.md"
	if b, err := os.ReadFile(path); err == nil {
		c.Data(200, "text/markdown; charset=utf-8", b)
		return
	}
	fallback := `---
name: rxui-control-discovery
description: Use runtime discovery endpoints instead of hardcoded action assumptions.
---

1. GET /api/v1/control/bootstrap
2. GET /api/v1/control/discovery
3. GET /api/v1/control/manifest
4. GET /api/v1/control/errors
5. Then call POST /api/v1/control/query or /exec with required ed25519 signature headers.
`
	c.Data(200, "text/markdown; charset=utf-8", []byte(fallback))
}

func handleControlListClients(c *gin.Context) {
	m := parseControlClients()
	arr := make([]controlClient, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": arr})
}

func handleControlAuditList(c *gin.Context) {
	limit := 50
	if v := intFromAny(c.Query("limit")); v > 0 && v <= 500 {
		limit = v
	}
	var logs []model.ControlAuditLog
	q := db.Order("id desc").Limit(limit)
	if clientID := strings.TrimSpace(c.Query("clientId")); clientID != "" {
		q = q.Where("client_id = ?", clientID)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		q = q.Where("action = ?", action)
	}
	q.Find(&logs)
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": logs})
}

func handleControlUpsertClient(c *gin.Context) {
	var req controlClient
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ClientID) == "" || strings.TrimSpace(req.PublicKey) == "" {
		c.JSON(400, gin.H{"code": 1, "message": "参数错误: 需要 clientId/publicKey"})
		return
	}
	if _, err := decodePubKey(req.PublicKey); err != nil {
		c.JSON(400, gin.H{"code": 1, "message": "publicKey 无效: " + err.Error()})
		return
	}
	m := parseControlClients()
	if !req.Enabled {
		req.Enabled = true
	}
	m[req.ClientID] = req
	saveControlClients(m)
	c.JSON(200, gin.H{"code": 0, "message": "保存成功", "data": req})
}

func handleControlDeleteClient(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(400, gin.H{"code": 1, "message": "clientId 不能为空"})
		return
	}
	m := parseControlClients()
	delete(m, id)
	saveControlClients(m)
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func handleControlExportClient(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(400, gin.H{"code": 1, "message": "clientId 不能为空"})
		return
	}
	m := parseControlClients()
	cc, ok := m[id]
	if !ok {
		c.JSON(404, gin.H{"code": 1, "message": "客户端不存在"})
		return
	}

	host := c.Request.Host
	if host == "" {
		port := strings.TrimSpace(settings["webPort"])
		if port == "" {
			port = "54321"
		}
		host = "127.0.0.1:" + port
	}
	scheme := "http"
	if c.Request.TLS != nil || (settings["webCertFile"] != "" && settings["webKeyFile"] != "") {
		scheme = "https"
	}
	basePath := strings.TrimRight(strings.TrimSpace(settings["webBasePath"]), "/")
	if basePath == "" {
		basePath = ""
	}
	
	config := gin.H{
		"clientId": cc.ClientID,
		"publicKey": cc.PublicKey,
		"algorithm": "Ed25519",
		"enabled": cc.Enabled,
		"remark": cc.Remark,
		"endpoints": gin.H{
			"bootstrap": fmt.Sprintf("%s://%s%s/api/v1/control/bootstrap", scheme, host, basePath),
			"discovery": fmt.Sprintf("%s://%s%s/api/v1/control/discovery", scheme, host, basePath),
			"manifest": fmt.Sprintf("%s://%s%s/api/v1/control/manifest", scheme, host, basePath),
			"errors": fmt.Sprintf("%s://%s%s/api/v1/control/errors", scheme, host, basePath),
			"query": fmt.Sprintf("%s://%s%s/api/v1/control/query", scheme, host, basePath),
			"exec": fmt.Sprintf("%s://%s%s/api/v1/control/exec", scheme, host, basePath),
			"audit": fmt.Sprintf("%s://%s%s/api/v1/control/audit", scheme, host, basePath),
		},
		"headers": gin.H{
			"X-Rxui-Client": "{{clientId}}",
			"X-Rxui-Timestamp": "{{timestamp}}",
			"X-Rxui-Nonce": "{{nonce}}",
			"X-Rxui-Signature": "{{signature}}",
		},
		"signatureMethod": "Ed25519",
		"signatureFormat": "Base64",
		"timestampWindow": nonceWindow,
		"idempotencyWindow": requestWindow,
		"hint": "AI 可以直接导入此配置，需要提供对应的私钥才能签名请求。",
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": config})
}

func verifyControlSignature(c *gin.Context) (string, string, bool, string) {
	clientID := strings.TrimSpace(c.GetHeader("X-Rxui-Client"))
	tsStr := strings.TrimSpace(c.GetHeader("X-Rxui-Timestamp"))
	nonce := strings.TrimSpace(c.GetHeader("X-Rxui-Nonce"))
	sigB64 := strings.TrimSpace(c.GetHeader("X-Rxui-Signature"))
	if clientID == "" || tsStr == "" || nonce == "" || sigB64 == "" {
		return "", "", false, "missing_signature_headers"
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", "", false, "invalid_timestamp"
	}
	now := time.Now().Unix()
	if ts < now-nonceWindow || ts > now+nonceWindow {
		return "", "", false, "timestamp_out_of_window"
	}

	m := parseControlClients()
	cc, ok := m[clientID]
	if !ok || !cc.Enabled {
		return "", "", false, "client_not_allowed"
	}
	pub, err := decodePubKey(cc.PublicKey)
	if err != nil {
		return "", "", false, "invalid_client_pubkey"
	}

	nonceKey := clientID + ":" + nonce
	nonceMu.Lock()
	if _, exists := seenNonces[nonceKey]; exists {
		nonceMu.Unlock()
		return "", "", false, "nonce_replayed"
	}
	seenNonces[nonceKey] = now
	for k, t := range seenNonces {
		if now-t > nonceWindow*2 {
			delete(seenNonces, k)
		}
	}
	nonceMu.Unlock()

	body, _ := c.GetRawData()
	c.Request.Body = ioNopCloser(body)
	h := sha256.Sum256(body)
	signText := strings.Join([]string{c.Request.Method, c.FullPath(), tsStr, nonce, hex.EncodeToString(h[:])}, "\n")
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return "", "", false, "invalid_signature_encoding"
	}
	if !ed25519.Verify(pub, []byte(signText), sig) {
		return "", "", false, "signature_verify_failed"
	}
	return clientID, hex.EncodeToString(h[:]), true, ""
}

func handleControlQuery(c *gin.Context) {
	started := time.Now()
	clientID, bodyHash, ok, reason := verifyControlSignature(c)
	if !ok {
		resp := gin.H{"ok": false, "error": gin.H{"code": reason}}
		auditControl(clientID, c.FullPath(), "", true, false, reason, "", "", bodyHash, time.Since(started))
		c.JSON(http.StatusUnauthorized, resp)
		return
	}
	var req actionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := gin.H{"ok": false, "error": gin.H{"code": "INVALID_JSON"}}
		auditControl(clientID, c.FullPath(), "", true, false, "INVALID_JSON", err.Error(), "", bodyHash, time.Since(started))
		c.JSON(400, resp)
		return
	}
	if cached, hit := loadCachedResponse(clientID, c.FullPath(), req.RequestID); hit {
		auditControl(clientID, c.FullPath(), req.Action, true, true, "", "", req.RequestID, bodyHash, time.Since(started))
		c.JSON(200, cached)
		return
	}

	resp := executeAction(req, true)
	resp["clientId"] = clientID
	if req.RequestID != "" {
		resp["requestId"] = req.RequestID
	}
	success, code, msg := parseRespStatus(resp)
	auditControl(clientID, c.FullPath(), req.Action, true, success, code, msg, req.RequestID, bodyHash, time.Since(started))
	storeCachedResponse(clientID, c.FullPath(), req.RequestID, resp)
	c.JSON(200, resp)
}

func handleControlExec(c *gin.Context) {
	started := time.Now()
	clientID, bodyHash, ok, reason := verifyControlSignature(c)
	if !ok {
		resp := gin.H{"ok": false, "error": gin.H{"code": reason}}
		auditControl(clientID, c.FullPath(), "", false, false, reason, "", "", bodyHash, time.Since(started))
		c.JSON(http.StatusUnauthorized, resp)
		return
	}
	var req actionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := gin.H{"ok": false, "error": gin.H{"code": "INVALID_JSON"}}
		auditControl(clientID, c.FullPath(), "", false, false, "INVALID_JSON", err.Error(), "", bodyHash, time.Since(started))
		c.JSON(400, resp)
		return
	}
	if cached, hit := loadCachedResponse(clientID, c.FullPath(), req.RequestID); hit {
		auditControl(clientID, c.FullPath(), req.Action, false, true, "", "", req.RequestID, bodyHash, time.Since(started))
		c.JSON(200, cached)
		return
	}

	resp := executeAction(req, false)
	resp["clientId"] = clientID
	if req.RequestID != "" {
		resp["requestId"] = req.RequestID
	}
	success, code, msg := parseRespStatus(resp)
	auditControl(clientID, c.FullPath(), req.Action, false, success, code, msg, req.RequestID, bodyHash, time.Since(started))
	storeCachedResponse(clientID, c.FullPath(), req.RequestID, resp)
	c.JSON(200, resp)
}

func controlCapabilities() gin.H {
	return gin.H{
		"query": []string{"xray.status", "sys.status", "inbound.list", "inbound.get", "client.list", "client.get", "cert.list", "cert.get", "logs.tail"},
		"exec":  []string{"xray.start", "xray.stop", "xray.restart", "inbound.create", "inbound.update", "inbound.delete", "client.create", "client.update", "client.delete", "cert.create", "cert.update", "cert.delete", "cert.renew", "net.ping"},
	}
}

func controlActionSpecs() []gin.H {
	return []gin.H{
		{"action": "xray.status", "mode": "query", "summary": "查询 Xray 运行状态", "params": gin.H{}},
		{"action": "sys.status", "mode": "query", "summary": "查询系统汇总状态", "params": gin.H{}},
		{"action": "inbound.list", "mode": "query", "summary": "列出入站", "params": gin.H{}},
		{"action": "inbound.get", "mode": "query", "summary": "获取单个入站", "params": gin.H{"id": "number(optional)", "tag": "string(optional)"}},
		{"action": "client.list", "mode": "query", "summary": "列出客户端", "params": gin.H{"inboundId": "number(optional)"}},
		{"action": "client.get", "mode": "query", "summary": "获取单个客户端", "params": gin.H{"id": "number(required)"}},
		{"action": "cert.list", "mode": "query", "summary": "列出证书", "params": gin.H{}},
		{"action": "cert.get", "mode": "query", "summary": "获取单个证书", "params": gin.H{"id": "number(optional)", "domain": "string(optional)"}},
		{"action": "logs.tail", "mode": "query", "summary": "读取面板日志", "params": gin.H{"lines": "number(optional,1-500)"}},
		{"action": "xray.start|stop|restart", "mode": "exec", "summary": "控制 Xray 进程", "params": gin.H{}},
		{"action": "inbound.create|update|delete", "mode": "exec", "summary": "入站增改删", "params": gin.H{"id": "number(update/delete required)"}},
		{"action": "client.create|update|delete", "mode": "exec", "summary": "客户端增改删", "params": gin.H{"id": "number(update/delete required)", "inboundId": "number(create required)"}},
		{"action": "cert.create|update|delete|renew", "mode": "exec", "summary": "证书管理", "params": gin.H{"id": "number(update/delete/renew required)", "domain": "string(create required)"}},
		{"action": "net.ping", "mode": "exec", "summary": "网络探测", "params": gin.H{"target": "string(required)"}},
	}
}

func controlErrorCodes() gin.H {
	return gin.H{
		"INVALID_JSON":               "请求体不是合法 JSON",
		"MISSING_ACTION":             "缺少 action 字段",
		"UNSUPPORTED_ACTION":         "不支持的 action",
		"QUERY_ONLY_ACTION":          "该 action 仅允许 query",
		"EXEC_ONLY_ACTION":           "该 action 仅允许 exec",
		"INVALID_PARAMS":             "参数缺失或格式错误",
		"NOT_FOUND":                  "目标资源不存在",
		"XRAY_ACTION_FAILED":         "Xray 启停失败",
		"XRAY_APPLY_FAILED":          "配置应用到 Xray 失败",
		"LOG_READ_FAILED":            "读取日志失败",
		"PING_FAILED":                "网络探测失败",
		"missing_signature_headers":  "鉴权请求头缺失",
		"invalid_timestamp":          "时间戳格式错误",
		"timestamp_out_of_window":    "时间戳超出允许窗口",
		"client_not_allowed":         "控制客户端未注册或被禁用",
		"invalid_client_pubkey":      "客户端公钥配置无效",
		"nonce_replayed":             "nonce 重放",
		"invalid_signature_encoding": "签名编码无效",
		"signature_verify_failed":    "签名校验失败",
	}
}

func executeAction(req actionReq, queryOnly bool) gin.H {
	action := strings.TrimSpace(req.Action)
	if action == "" {
		return gin.H{"ok": false, "error": gin.H{"code": "MISSING_ACTION"}}
	}
	out := gin.H{"ok": true, "action": action, "ts": time.Now().Unix()}
	errResp := func(code, msg string) gin.H {
		return gin.H{"ok": false, "action": action, "error": gin.H{"code": code, "message": msg}}
	}

	switch action {
	case "xray.status":
		out["data"] = gin.H{"running": xrayRunning}
	case "sys.status":
		var inbounds []model.Inbound
		db.Find(&inbounds)
		totalUp, totalDown := int64(0), int64(0)
		for _, i := range inbounds {
			totalUp += i.Up
			totalDown += i.Down
		}
		out["data"] = gin.H{"inboundCount": len(inbounds), "traffic": gin.H{"up": totalUp, "down": totalDown}}
	case "inbound.list":
		var inbounds []model.Inbound
		db.Order("id desc").Find(&inbounds)
		out["data"] = inbounds
	case "inbound.get":
		id := intFromAny(req.Params["id"])
		var inbound model.Inbound
		if id > 0 {
			if err := db.First(&inbound, id).Error; err != nil {
				return errResp("NOT_FOUND", "inbound not found")
			}
		} else {
			tag := strFromAny(req.Params["tag"])
			if tag == "" || db.Where("tag = ?", tag).First(&inbound).Error != nil {
				return errResp("NOT_FOUND", "inbound not found")
			}
		}
		out["data"] = inbound
	case "client.list":
		var clients []model.Client
		inboundID := intFromAny(req.Params["inboundId"])
		if inboundID > 0 {
			db.Where("inbound_id = ?", inboundID).Order("id desc").Find(&clients)
		} else {
			db.Order("id desc").Find(&clients)
		}
		out["data"] = clients
	case "client.get":
		id := intFromAny(req.Params["id"])
		var client model.Client
		if id <= 0 || db.First(&client, id).Error != nil {
			return errResp("NOT_FOUND", "client not found")
		}
		out["data"] = client
	case "cert.list":
		var certs []model.Certificate
		db.Order("id desc").Find(&certs)
		out["data"] = certs
	case "cert.get":
		id := intFromAny(req.Params["id"])
		var cert model.Certificate
		if id > 0 {
			if db.First(&cert, id).Error != nil {
				return errResp("NOT_FOUND", "certificate not found")
			}
		} else {
			domain := strFromAny(req.Params["domain"])
			if domain == "" || db.Where("domain = ?", domain).First(&cert).Error != nil {
				return errResp("NOT_FOUND", "certificate not found")
			}
		}
		out["data"] = cert
	case "logs.tail":
		if !queryOnly {
			return errResp("QUERY_ONLY_ACTION", "logs.tail is query-only")
		}
		n := intFromAny(req.Params["lines"])
		if n <= 0 || n > 500 {
			n = 120
		}
		cmd := exec.Command("bash", "-lc", fmt.Sprintf("journalctl -u rx-ui -n %d --no-pager", n))
		b, err := cmd.CombinedOutput()
		if err != nil {
			return errResp("LOG_READ_FAILED", err.Error())
		}
		out["data"] = gin.H{"logs": string(b)}
	case "xray.start", "xray.stop", "xray.restart":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "xray action is exec-only")
		}
		var err error
		if action == "xray.start" {
			err = startXray()
		}
		if action == "xray.stop" {
			err = stopXray()
		}
		if action == "xray.restart" {
			_ = stopXray()
			time.Sleep(200 * time.Millisecond)
			err = startXray()
		}
		if err != nil {
			return errResp("XRAY_ACTION_FAILED", err.Error())
		}
		out["data"] = gin.H{"running": xrayRunning}
	case "net.ping":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "net.ping is exec-only")
		}
		target := strFromAny(req.Params["target"])
		if target == "" {
			return errResp("MISSING_TARGET", "target is required")
		}
		cmd := exec.Command("ping", "-c", "4", "-W", "2", target)
		b, err := cmd.CombinedOutput()
		if err != nil && len(b) == 0 {
			return errResp("PING_FAILED", err.Error())
		}
		out["data"] = gin.H{"target": target, "result": string(b)}
	case "inbound.create":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "inbound.create is exec-only")
		}
		var inbound model.Inbound
		b, _ := json.Marshal(req.Params)
		if err := json.Unmarshal(b, &inbound); err != nil {
			return errResp("INVALID_PARAMS", err.Error())
		}
		if inbound.Tag == "" {
			inbound.Tag = fmt.Sprintf("inbound-%d", inbound.Port)
		}
		if err := db.Create(&inbound).Error; err != nil {
			return errResp("CREATE_FAILED", err.Error())
		}
		if err := applyInboundRuntimeChanges(); err != nil {
			_ = db.Delete(&model.Inbound{}, inbound.ID).Error
			return errResp("XRAY_APPLY_FAILED", err.Error())
		}
		out["data"] = inbound
	case "inbound.update":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "inbound.update is exec-only")
		}
		id := intFromAny(req.Params["id"])
		if id <= 0 {
			return errResp("MISSING_ID", "id is required")
		}
		var old model.Inbound
		if db.First(&old, id).Error != nil {
			return errResp("NOT_FOUND", "inbound not found")
		}
		inbound := old
		b, _ := json.Marshal(req.Params)
		_ = json.Unmarshal(b, &inbound)
		inbound.ID = old.ID
		if err := db.Save(&inbound).Error; err != nil {
			return errResp("UPDATE_FAILED", err.Error())
		}
		if err := applyInboundRuntimeChanges(); err != nil {
			_ = db.Save(&old).Error
			return errResp("XRAY_APPLY_FAILED", err.Error())
		}
		out["data"] = inbound
	case "inbound.delete":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "inbound.delete is exec-only")
		}
		id := intFromAny(req.Params["id"])
		if id <= 0 {
			return errResp("MISSING_ID", "id is required")
		}
		_ = db.Where("inbound_id = ?", id).Delete(&model.Client{}).Error
		_ = db.Delete(&model.Inbound{}, id).Error
		if err := applyInboundRuntimeChanges(); err != nil {
			return errResp("XRAY_APPLY_FAILED", err.Error())
		}
		out["data"] = gin.H{"deleted": id}
	case "client.create":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "client.create is exec-only")
		}
		var client model.Client
		b, _ := json.Marshal(req.Params)
		if err := json.Unmarshal(b, &client); err != nil || client.InboundID == 0 {
			return errResp("INVALID_PARAMS", "inboundId is required")
		}
		if err := db.Create(&client).Error; err != nil {
			return errResp("CREATE_FAILED", err.Error())
		}
		_ = syncInboundClientsToSettings(client.InboundID)
		out["data"] = client
	case "client.update":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "client.update is exec-only")
		}
		id := intFromAny(req.Params["id"])
		var client model.Client
		if id <= 0 || db.First(&client, id).Error != nil {
			return errResp("NOT_FOUND", "client not found")
		}
		inboundID := client.InboundID
		b, _ := json.Marshal(req.Params)
		_ = json.Unmarshal(b, &client)
		client.ID = id
		if err := db.Save(&client).Error; err != nil {
			return errResp("UPDATE_FAILED", err.Error())
		}
		_ = syncInboundClientsToSettings(inboundID)
		out["data"] = client
	case "client.delete":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "client.delete is exec-only")
		}
		id := intFromAny(req.Params["id"])
		var client model.Client
		if id <= 0 || db.First(&client, id).Error != nil {
			return errResp("NOT_FOUND", "client not found")
		}
		inboundID := client.InboundID
		_ = db.Delete(&model.Client{}, id).Error
		_ = syncInboundClientsToSettings(inboundID)
		out["data"] = gin.H{"deleted": id}
	case "cert.create":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "cert.create is exec-only")
		}
		var cert model.Certificate
		b, _ := json.Marshal(req.Params)
		if err := json.Unmarshal(b, &cert); err != nil || strings.TrimSpace(cert.Domain) == "" {
			return errResp("INVALID_PARAMS", "domain is required")
		}
		if err := ensureCertificateFiles(&cert); err != nil {
			return errResp("CERT_FILE_FAILED", err.Error())
		}
		fillCertMeta(&cert)
		if err := db.Create(&cert).Error; err != nil {
			return errResp("CREATE_FAILED", err.Error())
		}
		out["data"] = cert
	case "cert.update":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "cert.update is exec-only")
		}
		id := intFromAny(req.Params["id"])
		var cert model.Certificate
		if id <= 0 || db.First(&cert, id).Error != nil {
			return errResp("NOT_FOUND", "certificate not found")
		}
		old := cert
		b, _ := json.Marshal(req.Params)
		_ = json.Unmarshal(b, &cert)
		cert.ID = old.ID
		if cert.Domain == "" {
			cert.Domain = old.Domain
		}
		if err := ensureCertificateFiles(&cert); err != nil {
			return errResp("CERT_FILE_FAILED", err.Error())
		}
		fillCertMeta(&cert)
		if err := db.Save(&cert).Error; err != nil {
			return errResp("UPDATE_FAILED", err.Error())
		}
		out["data"] = cert
	case "cert.delete":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "cert.delete is exec-only")
		}
		id := intFromAny(req.Params["id"])
		if id <= 0 {
			return errResp("MISSING_ID", "id is required")
		}
		_ = db.Delete(&model.Certificate{}, id).Error
		out["data"] = gin.H{"deleted": id}
	case "cert.renew":
		if queryOnly {
			return errResp("EXEC_ONLY_ACTION", "cert.renew is exec-only")
		}
		id := intFromAny(req.Params["id"])
		var cert model.Certificate
		if id <= 0 || db.First(&cert, id).Error != nil {
			return errResp("NOT_FOUND", "certificate not found")
		}
		if err := renewCertificate(&cert); err != nil {
			return errResp("RENEW_FAILED", err.Error())
		}
		out["data"] = cert
	default:
		return errResp("UNSUPPORTED_ACTION", action)
	}

	return out
}

func cloneResp(resp gin.H) gin.H {
	b, _ := json.Marshal(resp)
	var out gin.H
	_ = json.Unmarshal(b, &out)
	return out
}

func idempotencyKey(clientID, path, requestID string) string {
	return clientID + "|" + path + "|" + strings.TrimSpace(requestID)
}

func loadCachedResponse(clientID, path, requestID string) (gin.H, bool) {
	if strings.TrimSpace(requestID) == "" {
		return nil, false
	}
	now := time.Now().Unix()
	requestMu.Lock()
	defer requestMu.Unlock()
	for k, v := range requestSeen {
		if now-v.At > requestWindow {
			delete(requestSeen, k)
		}
	}
	k := idempotencyKey(clientID, path, requestID)
	v, ok := requestSeen[k]
	if !ok {
		return nil, false
	}
	resp := cloneResp(v.Resp)
	resp["idempotencyHit"] = true
	return resp, true
}

func storeCachedResponse(clientID, path, requestID string, resp gin.H) {
	if strings.TrimSpace(requestID) == "" {
		return
	}
	requestMu.Lock()
	requestSeen[idempotencyKey(clientID, path, requestID)] = struct {
		At   int64
		Resp gin.H
	}{At: time.Now().Unix(), Resp: cloneResp(resp)}
	requestMu.Unlock()
}

func parseRespStatus(resp gin.H) (bool, string, string) {
	okVal, _ := resp["ok"].(bool)
	if okVal {
		return true, "", ""
	}
	errMap, _ := resp["error"].(gin.H)
	if errMap == nil {
		if m, ok := resp["error"].(map[string]interface{}); ok {
			errMap = gin.H(m)
		}
	}
	code, _ := errMap["code"].(string)
	msg, _ := errMap["message"].(string)
	return false, code, msg
}

func auditControl(clientID, path, action string, queryOnly, success bool, errCode, errMsg, requestID, bodyHash string, d time.Duration) {
	_ = db.Create(&model.ControlAuditLog{
		ClientID:   clientID,
		Path:       path,
		Action:     action,
		QueryOnly:  queryOnly,
		Success:    success,
		ErrorCode:  errCode,
		ErrorMsg:   errMsg,
		RequestID:  requestID,
		BodySHA256: bodyHash,
		DurationMs: d.Milliseconds(),
	}).Error
}

func decodePubKey(s string) (ed25519.PublicKey, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty")
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b), nil
	}
	if b, err := hex.DecodeString(s); err == nil && len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b), nil
	}
	return nil, fmt.Errorf("expect base64/hex ed25519 public key")
}

type rawBodyCloser struct{ *strings.Reader }

func (r rawBodyCloser) Close() error { return nil }

func ioNopCloser(b []byte) rawBodyCloser {
	return rawBodyCloser{Reader: strings.NewReader(string(b))}
}

func intFromAny(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(x))
		return i
	default:
		return 0
	}
}

func strFromAny(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}
