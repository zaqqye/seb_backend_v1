package middleware

import (
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "gorm.io/gorm"

    "github.com/example/seb_backend_v1/internal/models"
)

type AuthConfig struct {
    JWTSecret    string
    JWTExpiresIn time.Duration
}

type Claims struct {
    UserID  string `json:"user_id"`
    Role    string `json:"role"`
    Email   string `json:"email"`
    jwt.RegisteredClaims
}

func AuthMiddleware(db *gorm.DB, cfg AuthConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        auth := c.GetHeader("Authorization")
        if auth == "" || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
            return
        }
        tokenStr := strings.TrimSpace(auth[len("Bearer "):])

        claims := &Claims{}
        token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
            return []byte(cfg.JWTSecret), nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }

        var user models.User
        if err := db.Where("user_id = ? AND active = ?", claims.UserID, true).First(&user).Error; err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found or inactive"})
            return
        }

        c.Set("user", user)
        c.Next()
    }
}

func RequireRoles(roles ...string) gin.HandlerFunc {
    allowed := map[string]struct{}{}
    for _, r := range roles {
        allowed[r] = struct{}{}
    }
    return func(c *gin.Context) {
        uVal, ok := c.Get("user")
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }
        user := uVal.(models.User)
        if _, ok := allowed[user.Role]; !ok {
            // allow admin to pass any role-gate
            if user.Role != "admin" {
                c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
                return
            }
        }
        c.Next()
    }
}

