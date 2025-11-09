package controllers

import (
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type OAuthController struct {
    Cfg *config.Config
}

type moodleSSOResponse struct {
    SSOURL string `json:"sso_url"`
    Token  string `json:"token"`
}

// GenerateMoodleSSO issues a short-lived JWT that Moodle can validate for auto-login.
// Requires authenticated user (admin/pengawas/siswa) and configured secrets.
func (oc *OAuthController) GenerateMoodleSSO(c *gin.Context) {
    if oc == nil || oc.Cfg == nil || oc.Cfg.MoodleSSOSecret == "" || oc.Cfg.MoodleSSOLoginURL == "" {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "moodle sso not configured"})
        return
    }
    uVal, ok := c.Get("user")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    user := uVal.(models.User)
    if user.Email == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "user has no email"})
        return
    }

    expiresAt := time.Now().Add(2 * time.Minute)
    claims := jwt.MapClaims{
        "sub":   user.ID,
        "email": user.Email,
        "name":  user.FullName,
        "role":  user.Role,
        "exp":   expiresAt.Unix(),
        "iat":   time.Now().Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString([]byte(oc.Cfg.MoodleSSOSecret))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
        return
    }

    redirect := c.DefaultQuery("redirect", oc.Cfg.MoodleSSOLoginURL)
    ssoURL := fmt.Sprintf("%s?token=%s", redirect, signed)

    c.JSON(http.StatusOK, moodleSSOResponse{
        SSOURL: ssoURL,
        Token:  signed,
    })
}
