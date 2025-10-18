package middleware

import (
	"aqlesia/platform/logger"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GinLogger(log logger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		query := ctx.Request.URL.RawQuery
		id := uuid.New().String()
		ctx.Set("x-request-id", id)
		ctx.Set("request-start-time", start)
		ctx.Next()

		end := time.Now()
		latency := end.Sub(start)
		fields := []zapcore.Field{
			zap.Int("status", ctx.Writer.Status()),
			zap.String("method", ctx.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", ctx.ClientIP()),
			zap.String("user-agent", ctx.Request.UserAgent()),
			zap.Int64("request-latency", latency.Milliseconds()),
		}
		log.Info(ctx, "GIN", fields...)
	}
}
