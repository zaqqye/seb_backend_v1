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

    switch name {
    case "login":
        payload := gin.H{
            "schema_version": 1,
            "screen_version": 1,
            "name":           "login",
            "platform":       platform,
            "components": []gin.H{
                {"id": "title", "type": "text", "props": gin.H{"text": "Masuk", "style": "h1"}},
                {"id": "email", "type": "input", "props": gin.H{"name": "email", "hint": "Email", "keyboard": "email"}},
                {"id": "password", "type": "input", "props": gin.H{"name": "password", "hint": "Password", "secure": true}},
                {"id": "loginBtn", "type": "button",
                    "props": gin.H{"text": "Login", "style": "primary"},
                    "action": gin.H{
                        "type":   "http",
                        "method": "POST",
                        "url":    "/api/v1/auth/login",
                        "bodyBindings": gin.H{"email": "email", "password": "password"},
                        "onSuccess": []gin.H{
                            {"type": "storeToken", "from": "response.access_token"},
                            {"type": "navigate", "to": "dashboard"},
                        },
                        "onError": []gin.H{{"type": "toast", "message": "Email/Password salah"}},
                    },
                },
            },
        }
        s.respondWithSignature(c, payload)
        return
    case "dashboard":
        if role == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }
        title := "Dashboard"
        if role != "" {
            title = "Dashboard (" + role + ")"
        }
        payload := gin.H{
            "schema_version": 1,
            "screen_version": 1,
            "name":           "dashboard",
            "platform":       platform,
            "components": []gin.H{
                {"id": "title", "type": "text", "props": gin.H{"text": title, "style": "h1"}},
                {"id": "me", "type": "button",
                    "props": gin.H{"text": "Profil Saya"},
                    "action": gin.H{"type": "http", "auth": "bearer", "method": "GET", "url": "/api/v1/auth/me"},
                },
            },
        }
        s.respondWithSignature(c, payload)
        return
    case "exit_code":
        if role == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }
        payload := gin.H{
            "schema_version": 1,
            "screen_version": 1,
            "name":           "exit_code",
            "platform":       platform,
            "components": []gin.H{
                {"id": "code", "type": "input", "props": gin.H{"name": "code", "hint": "Masukkan Exit Code"}},
                {"id": "submit", "type": "button",
                    "props": gin.H{"text": "Gunakan Kode"},
                    "action": gin.H{
                        "type":   "http",
                        "method": "POST",
                        "url":    "/api/v1/exit-codes/consume",
                        "auth":   "bearer",
                        "bodyBindings": gin.H{"code": "code"},
                        "onSuccess": []gin.H{{"type": "toast", "message": "Kode diterima"}, {"type": "navigate", "to": "dashboard"}},
                        "onError":   []gin.H{{"type": "toast", "message": "Kode tidak valid/terpakai"}},
                    },
                },
            },
        }
        s.respondWithSignature(c, payload)
        return
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
    if sec := strings.TrimSpace(s.Cfg.SDUIHMACSecret); sec != "" {
        mac := hmac.New(sha256.New, []byte(sec))
        mac.Write(raw)
        sig := mac.Sum(nil)
        c.Header("X-SDUI-Signature", hex.EncodeToString(sig))
    }
    c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}
