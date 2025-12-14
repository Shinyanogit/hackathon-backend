package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	authClient *auth.Client
}

func NewAuthMiddleware(ctx context.Context) (*AuthMiddleware, error) {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "FIREBASE_PROJECT_ID is not set")
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
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

func (m *AuthMiddleware) Client() *auth.Client {
	return m.authClient
}
