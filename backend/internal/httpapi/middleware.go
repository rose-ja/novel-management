package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 64 {
			buf := make([]byte, 8)
			if _, err := rand.Read(buf); err != nil {
				id = "unknown"
			} else {
				id = hex.EncodeToString(buf)
			}
		}
		c.Set(requestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetString(requestIDKey),
		)
	}
}

func securityHeaders() gin.HandlerFunc {
	// Next.js 静态导出包含内联 hydration 脚本，故 script-src 允许 'unsafe-inline'。
	const csp = "default-src 'self'; base-uri 'none'; img-src 'self' data:; " +
		"font-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; " +
		"connect-src 'self'; frame-ancestors 'none'; form-action 'self'"

	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

func hostGuard(allowed []string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, host := range allowed {
		allowedSet[strings.ToLower(host)] = struct{}{}
	}

	return func(c *gin.Context) {
		if _, ok := allowedSet[strings.ToLower(c.Request.Host)]; !ok {
			abortError(c, http.StatusForbidden, "HOST_NOT_ALLOWED", "请求来源不在 Host 白名单内")
			return
		}
		c.Next()
	}
}

func recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic",
					"error", rec,
					"request_id", c.GetString(requestIDKey),
					"path", c.Request.URL.Path,
				)
				abortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
			}
		}()
		c.Next()
	}
}
