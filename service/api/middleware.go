package api

import (
	"slices"
	"strings"

	"github.com/labstack/echo/v4"
)

func CreateAPIMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := &APIContext{
				Context: c,
			}

			return next(ctx)
		}
	}
}

func CreateAPIV1Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := GetAPIContext(c)

			header := c.Request().Header
			req := c.Request()
			path := req.URL.Path

			authorization := header.Get("Authorization")
			if authorization == "" {
				return ctx.RespNoAuthMsg("no authorization header")
			}

			apikey := strings.TrimSpace(strings.ReplaceAll(authorization, "Bearer", ""))
			if apikey == "" {
				return ctx.RespNoAuthMsg("no authorization key")
			}

			serverKeyPaths := []string{
				"/v1/list-tools",
				"/v1/call-tool",
			}

			if slices.Contains(serverKeyPaths, path) {

			} else {
				// todo: check access key
			}

			return next(ctx)
		}
	}
}
