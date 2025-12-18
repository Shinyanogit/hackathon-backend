package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

type AuthMiddleware struct {
	authClient      *auth.Client
	projectFromEnv  string
	projectFromCred string
}

func NewAuthMiddleware(ctx context.Context) (*AuthMiddleware, error) {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "FIREBASE_PROJECT_ID is not set")
	}

	creds, _ := google.FindDefaultCredentials(ctx)
	credProject := ""
	if creds != nil {
		credProject = creds.ProjectID
	}

	var opts []option.ClientOption
	if jsonStr := os.Getenv("FIREBASE_CREDENTIALS_JSON"); jsonStr != "" {
		decoded, err := base64.StdEncoding.DecodeString(jsonStr)
		if err != nil {
			// if not base64, try raw json
			decoded = []byte(jsonStr)
		}
		opts = append(opts, option.WithCredentialsJSON(decoded))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, opts...)
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}
	return &AuthMiddleware{
		authClient:      client,
		projectFromEnv:  projectID,
		projectFromCred: credProject,
	}, nil
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

func (m *AuthMiddleware) EnvProjectID() string {
	return m.projectFromEnv
}

func (m *AuthMiddleware) CredProjectID() string {
	return m.projectFromCred
}
