package middleware

import (
	"context"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	authClient *auth.Client
}

func NewAuthMiddleware(ctx context.Context) (*AuthMiddleware, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}
	return &AuthMiddleware{authClient: client}, nil
}

func (m *AuthMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authz := c.Request().Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		tokenStr := strings.TrimPrefix(authz, "Bearer ")
		token, err := m.authClient.VerifyIDToken(c.Request().Context(), tokenStr)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		}
		c.Set("uid", token.UID)
		return next(c)
	}
}
