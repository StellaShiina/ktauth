package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/StellaShiina/ktauth/internal/logctx"
	"github.com/gin-gonic/gin"
)

// AccessLog logs a single audit line for every 4xx/5xx response.
// 2xx (including 204) responses are silent by design.
// c.Writer.Status() is safe to read after c.Next(): gin's responseWriter
// defaults to 200 and every status-setting path updates it before the
// handler chain returns.
// The cause of the status is read from the audit-log context recorded by the
// middleware or handler that produced it (internal/logctx).
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		if status < 400 {
			return
		}

		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"host", c.Request.Host,
			"clientIP", c.ClientIP(),
			"durationMs", float64(time.Since(start).Microseconds()) / 1000,
		}

		// TODO: Customizable X-KEY
		// Original client request as reported by the reverse proxy:
		// nginx auth_request sets X-Original-URI / X-Original-Method,
		// Caddy forward_auth sets X-Forwarded-Uri / X-Forwarded-Method.
		if v := c.GetHeader("X-Original-URI"); v != "" {
			attrs = append(attrs, "originalURI", v)
		}
		if v := c.GetHeader("X-Original-Method"); v != "" {
			attrs = append(attrs, "originalMethod", v)
		}
		if c.GetHeader("X-Original-URI") == "" {
			if v := c.GetHeader("X-Forwarded-Uri"); v != "" {
				attrs = append(attrs, "originalURI", v)
			}
			if v := c.GetHeader("X-Forwarded-Method"); v != "" {
				attrs = append(attrs, "originalMethod", v)
			}
		}

		// ACL rule context (whitelist/blacklist/greylist) set by CheckIPMiddleware.
		if rule := c.GetString(logctx.KeyRule); rule != "" {
			attrs = append(attrs, "rule", rule)
		}

		// Audit classification: the specific cause recorded by whichever
		// middleware or handler produced this status. Sites that record none
		// are reported as unattributed, which keeps the gaps greppable.
		reason := logctx.ReasonOf(c)
		if reason == "" {
			reason = string(logctx.ReasonUnattributed)
			if status == http.StatusNotFound {
				// gin's serveError: no route matched this method and path.
				// HandleMethodNotAllowed is off, so a method mismatch is a 404.
				reason = string(logctx.ReasonRouteNotFound)
			}
		}
		attrs = append(attrs, "reason", reason)

		if detail := logctx.DetailOf(c); detail != "" {
			attrs = append(attrs, "detail", detail)
		}

		// Authenticated actor, set by AuthMiddleWare once the session verifies.
		if uuid := c.GetString("uuid"); uuid != "" {
			attrs = append(attrs, "uuid", uuid)
		}

		if status >= 500 {
			slog.Error("request failed", attrs...)
		} else {
			slog.Warn("request rejected", attrs...)
		}
	}
}

// SlogWriter adapts an io.Writer to slog. It is used to redirect gin's
// debug output (DefaultWriter) and panic logs (DefaultErrorWriter) into
// the configured slog logger at the given level. Multi-line inputs (e.g.
// panic stack traces) are emitted as one slog record per line.
type SlogWriter struct {
	level slog.Level
}

func NewSlogWriter(level slog.Level) *SlogWriter {
	return &SlogWriter{level: level}
}

func (w *SlogWriter) Write(p []byte) (int, error) {
	for line := range strings.SplitSeq(string(p), "\n") {
		// gin Recovery colors panic output with ANSI codes; strip them.
		line = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(line, "\x1b[31m", ""), "\x1b[0m", ""))
		if line == "" {
			continue
		}
		slog.Log(context.Background(), w.level, line)
	}
	return len(p), nil
}

// ensure io.Writer is satisfied at compile time
var _ io.Writer = (*SlogWriter)(nil)
