//nolint:nestif // this is third party code
package middleware

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"aqlesia/platform/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecoveryWithZap returns a gin.HandlerFunc (middleware) that recovers from any panics and logs the panic to zap.
// It returns a 500 Internal Server response.
//
// It is a slightly modified version of ginzap.RecoveryWithZap()
func RecoveryWithZap(logger logger.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false) //nolint: errcheck // impossible to recover
				if brokenPipe {
					logger.Error(c, c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					_ = c.Error(err.(error))
					c.Abort()
					return
				}

				if stack {
					logger.Error(c, "[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					logger.Error(c, "[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"ok": false,
					"error": gin.H{
						"code":    http.StatusInternalServerError,
						"message": "Unexpected Internal Server Error. Please contact the administrator.",
					},
				})
			}
		}()
		c.Next()
	}
}
