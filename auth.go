package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 会话令牌有效期
const authTokenTTL = 7 * 24 * time.Hour

// ensureAuthSecret 确保存在用于签发会话令牌的服务端密钥（首次启动随机生成并持久化）
func ensureAuthSecret() string {
	s := strings.TrimSpace(settings["authSecret"])
	if s == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		s = hex.EncodeToString(b)
		settings["authSecret"] = s
		upsertSetting("authSecret", s)
	}
	return s
}

func signAuthMsg(msg string) string {
	mac := hmac.New(sha256.New, []byte(ensureAuthSecret()))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// makeAuthToken 生成 "username|expiryUnix|hmac" 形式的有状态无关会话令牌
func makeAuthToken(username string) string {
	msg := username + "|" + strconv.FormatInt(time.Now().Add(authTokenTTL).Unix(), 10)
	return msg + "|" + signAuthMsg(msg)
}

// validateAuthToken 校验令牌签名与有效期，返回用户名与是否有效
func validateAuthToken(token string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(token), "|")
	if len(parts) != 3 {
		return "", false
	}
	username, expStr, sig := parts[0], parts[1], parts[2]
	expected := signAuthMsg(username + "|" + expStr)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return "", false
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}
	return username, true
}

// 无需登录即可访问的路径（按 gin 路由模式 c.FullPath() 匹配）：
// - health / 登录
// - 订阅（供客户端 App 拉取；TODO: 后续应改为 per-subscription token 鉴权）
// - AI 控制端口的发现文档（不含密钥信息）与签名保护的 query/exec（自带 ed25519 鉴权）
var authPublicPaths = map[string]bool{
	"/api/v1/health":            true,
	"/api/v1/auth/login":        true,
	"/api/v1/sub":               true,
	"/api/v1/control/bootstrap": true,
	"/api/v1/control/discovery": true,
	"/api/v1/control/manifest":  true,
	"/api/v1/control/errors":    true,
	"/api/v1/control/skill":     true,
	"/api/v1/control/query":     true,
	"/api/v1/control/exec":      true,
}

// authMiddleware 对 /api/v1 下的管理接口强制校验登录令牌（白名单除外）
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if authPublicPaths[c.FullPath()] {
			c.Next()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(c.GetHeader("Authorization")), "Bearer "))
		username, ok := validateAuthToken(token)
		if !ok {
			c.JSON(401, gin.H{"code": 1, "message": "未登录或登录已过期"})
			c.Abort()
			return
		}
		c.Set("username", username)
		c.Next()
	}
}
