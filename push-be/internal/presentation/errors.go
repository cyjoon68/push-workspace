package presentation

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
)

func requestID(c echo.Context) string {
	if v, ok := c.Get("requestId").(string); ok {
		return v
	}
	return ""
}

func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	var de *domain.Error
	if errors.As(err, &de) {
		_ = c.JSON(de.Status, map[string]any{"error": map[string]any{
			"code":      de.Code,
			"message":   de.Message,
			"requestId": requestID(c),
			"details":   de.Details,
		}})
		return
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		code := domain.CodeValidation
		status := he.Code
		switch he.Code {
		case http.StatusNotFound:
			code = domain.CodeNotFound
		case http.StatusMethodNotAllowed:
			code = domain.CodeNotFound
		case http.StatusUnauthorized:
			code = domain.CodeUnauthenticated
		}
		_ = c.JSON(status, map[string]any{"error": map[string]any{
			"code":      code,
			"message":   http.StatusText(status),
			"requestId": requestID(c),
			"details":   nil,
		}})
		return
	}
	_ = c.JSON(http.StatusInternalServerError, map[string]any{"error": map[string]any{
		"code":      domain.CodeInternal,
		"message":   "internal error",
		"requestId": requestID(c),
		"details":   nil,
	}})
}
