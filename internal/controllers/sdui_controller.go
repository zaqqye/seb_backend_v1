package controllers

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type SDUIController struct {
    DB  *gorm.DB
    Cfg *config.Config
}

// GetScreen returns simple role-aware SDUI screen JSON.
func (s *SDUIController) GetScreen(c *gin.Context) {
    name := strings.ToLower(c.Param("name"))
    platform := strings.ToLower(c.DefaultQuery("platform", "android"))

    var role string
    if u, ok := c.Get("user"); ok {
        role = u.(models.User).Role
    }

    // Try dynamic SDUI from DB first
    var scr models.SduiScreen
    // prefer exact role match then fallback to role=""
    if err := s.DB.Where("name = ? AND platform = ? AND active = ? AND role IN ?",
        name, platform, true, []string{role, ""}).
        Order(s.DB.Statement.Quote("role")+" DESC"). // non-empty role first
        Order("screen_version DESC").
        First(&scr).Error; err == nil {
        if len(scr.Payload) > 0 {
            s.respondRawWithSignature(c, scr.Payload)
            return
        }
    }

    s.respondWithSignature(c, gin.H{"error": "screen not found"})
}

func (s *SDUIController) respondWithSignature(c *gin.Context, payload any) {
    // Marshal payload
    b, err := json.Marshal(payload)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode payload"})
        return
    }
    if u, ok := currentUser(c); ok {
        b = applyUserPlaceholders(b, u)
    }
    // Add HMAC signature header if secret provided
    if sec := strings.TrimSpace(s.Cfg.SDUIHMACSecret); sec != "" {
        mac := hmac.New(sha256.New, []byte(sec))
        mac.Write(b)
        sig := mac.Sum(nil)
        c.Header("X-SDUI-Signature", hex.EncodeToString(sig))
    }
    c.Data(http.StatusOK, "application/json; charset=utf-8", b)
}

func (s *SDUIController) respondRawWithSignature(c *gin.Context, raw []byte) {
    if u, ok := currentUser(c); ok {
        raw = applyUserPlaceholders(raw, u)
    }
    if sec := strings.TrimSpace(s.Cfg.SDUIHMACSecret); sec != "" {
        mac := hmac.New(sha256.New, []byte(sec))
        mac.Write(raw)
        sig := mac.Sum(nil)
        c.Header("X-SDUI-Signature", hex.EncodeToString(sig))
    }
    c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

func currentUser(c *gin.Context) (*models.User, bool) {
    if uVal, ok := c.Get("user"); ok {
        u := uVal.(models.User)
        return &u, true
    }
    return nil, false
}

func applyUserPlaceholders(data []byte, u *models.User) []byte {
    if u == nil {
        return data
    }
    s := string(data)
    r := strings.NewReplacer(
        "{{full_name}}", u.FullName,
        "{{email}}", u.Email,
        "{{role}}", u.Role,
        "{{user_id}}", u.ID,
    )
    s = r.Replace(s)
    return []byte(s)
}
