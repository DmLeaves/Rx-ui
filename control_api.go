package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
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

var (
	nonceMu     sync.Mutex
	seenNonces        = map[string]int64{}
	nonceWindow int64 = 120
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
			"type":      "ed25519-signature",
			"headers":   []string{"X-Rxui-Client", "X-Rxui-Timestamp", "X-Rxui-Nonce", "X-Rxui-Signature"},
			"signText":  "METHOD\\nPATH\\nTIMESTAMP\\nNONCE\\nSHA256_HEX(BODY)",
			"windowSec": nonceWindow,
		},
		"endpoints": gin.H{
			"bootstrap": "/api/v1/control/bootstrap",
			"query":     "/api/v1/control/query",
			"exec":      "/api/v1/control/exec",
		},
		"capabilities": controlCapabilities(),
		"examples": gin.H{
			"query": gin.H{"requestId": "req-1", "action": "xray.status", "params": gin.H{}},
			"exec":  gin.H{"requestId": "req-2", "action": "xray.restart", "params": gin.H{}},
		},
	}})
}

func handleControlListClients(c *gin.Context) {
	m := parseControlClients()
	arr := make([]controlClient, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": arr})
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
	resp := executeAction(req, true)
	resp["clientId"] = clientID
	if req.RequestID != "" {
		resp["requestId"] = req.RequestID
	}
	success, code, msg := parseRespStatus(resp)
	auditControl(clientID, c.FullPath(), req.Action, true, success, code, msg, req.RequestID, bodyHash, time.Since(started))
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
	resp := executeAction(req, false)
	resp["clientId"] = clientID
	if req.RequestID != "" {
		resp["requestId"] = req.RequestID
	}
	success, code, msg := parseRespStatus(resp)
	auditControl(clientID, c.FullPath(), req.Action, false, success, code, msg, req.RequestID, bodyHash, time.Since(started))
	c.JSON(200, resp)
}

func controlCapabilities() gin.H {
	return gin.H{
		"query": []string{"xray.status", "sys.status", "inbound.list", "inbound.get", "client.list", "client.get", "cert.list", "cert.get", "logs.tail"},
		"exec":  []string{"xray.start", "xray.stop", "xray.restart", "inbound.create", "inbound.update", "inbound.delete", "client.create", "client.update", "client.delete", "cert.create", "cert.update", "cert.delete", "cert.renew", "net.ping"},
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
