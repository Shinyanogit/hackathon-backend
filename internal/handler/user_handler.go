package handler

import (
	"net/http"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	authClient  *auth.Client
	envProject  string
	credProject string
}

func NewUserHandler(client *auth.Client, envProject, credProject string) *UserHandler {
	return &UserHandler{authClient: client, envProject: envProject, credProject: credProject}
}

type PublicUserResponse struct {
	UID         string  `json:"uid"`
	DisplayName string  `json:"displayName"`
	PhotoURL    *string `json:"photoURL"`
}

func (h *UserHandler) GetPublic(c echo.Context) error {
	uid := c.Param("uid")
	if uid == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid uid"))
	}
	user, err := h.authClient.GetUser(c.Request().Context(), uid)
	if err != nil {
		c.Logger().Warnf("get public user failed: uid=%s env_project=%s cred_project=%s err=%v", uid, h.envProject, h.credProject, err)
		return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "user not found"))
	}
	// 読み取り時に displayName/photoURL を確実に使用
	resp := PublicUserResponse{
		UID:         user.UID,
		DisplayName: user.DisplayName,
		PhotoURL:    strPtrOrNil(user.PhotoURL),
	}
	return c.JSON(http.StatusOK, resp)
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
