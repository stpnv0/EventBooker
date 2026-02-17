package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"
)

func Recovery(log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.LogAttrs(c.Request.Context(), logger.ErrorLevel, "panic recovered",
					logger.Any("error", err),
					logger.String("stack", string(debug.Stack())),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					ginext.H{"error": "internal server error"},
				)
			}
		}()

		c.Next()
	}
}