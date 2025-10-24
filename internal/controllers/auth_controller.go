package controllers

import (
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/middleware"
    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/utils"
)

type AuthController struct {
    DB        *gorm.DB
    AccessSecret string
    RefreshSecret string
    AccessTTL   time.Duration
    RefreshTTL  time.Duration
}

type registerRequest struct {
    FullName string `json:"full_name" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
    Kelas    string `json:"kelas"`
    Jurusan  string `json:"jurusan"`
    Role     string `json:"role"`   // admin-only endpoint will validate
    Active   *bool  `json:"active"` // optional, defaults to true
}

type loginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

func (a *AuthController) Register(c *gin.Context) {
    var req registerRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    pw, err := utils.HashPassword(req.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
        return
    }

    // Determine role (default to siswa if not provided)
    role := req.Role
    if role == "" {
        role = "siswa"
    }
    if !IsValidRole(role) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
        return
    }

    // Determine active flag (default true)
    active := true
    if req.Active != nil {
        active = *req.Active
    }

    user := models.User{
        UserID:   uuid.NewString(),
        FullName: req.FullName,
        Email:    req.Email,
        Password: pw,
        Role:     role,
        Kelas:    req.Kelas,
        Jurusan:  req.Jurusan,
        Active:   active,
    }

    if err := a.DB.Create(&user).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message":  "registered",
        "user_id":  user.UserID,
        "email":    user.Email,
        "full_name": user.FullName,
        "role":     user.Role,
    })
}

func (a *AuthController) Login(c *gin.Context) {
    var req loginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    if err := a.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    if !user.Active || !utils.CheckPassword(user.Password, req.Password) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    access, refresh, err := a.issueTokens(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "access_token": access.Token,
        "token_type":   "Bearer",
        "expires_in":   int(a.AccessTTL.Seconds()),
        "role":         user.Role,
        "refresh_token": refresh.Token,
        "refresh_expires_in": int(a.RefreshTTL.Seconds()),
    })
}

func (a *AuthController) Me(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)
    c.JSON(http.StatusOK, gin.H{
        "user_id":   user.UserID,
        "email":     user.Email,
        "full_name": user.FullName,
        "role":      user.Role,
        "kelas":     user.Kelas,
        "jurusan":   user.Jurusan,
        "active":    user.Active,
        "created_at": user.CreatedAt,
        "updated_at": user.UpdatedAt,
    })
}

type tokenPair struct {
    Token string
    JTI   string
}

func (a *AuthController) issueTokens(user models.User) (access tokenPair, refresh tokenPair, err error) {
    now := time.Now().UTC()
    sub := strconv.FormatUint(uint64(user.ID), 10)
    // Access token
    acl := middleware.Claims{
        UserID: user.UserID,
        Role:   user.Role,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    "seb_backend_v1",
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(a.AccessTTL)),
            Subject:   sub,
        },
    }
    at := jwt.NewWithClaims(jwt.SigningMethodHS256, acl)
    atStr, err := at.SignedString([]byte(a.AccessSecret))
    if err != nil { return }
    access = tokenPair{Token: atStr}

    // Refresh token with JTI
    jti := uuid.NewString()
    rcl := jwt.RegisteredClaims{
        Issuer:    "seb_backend_v1",
        IssuedAt:  jwt.NewNumericDate(now),
        ExpiresAt: jwt.NewNumericDate(now.Add(a.RefreshTTL)),
        Subject:   sub,
        ID:        jti,
    }
    rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rcl)
    rtStr, err := rt.SignedString([]byte(a.RefreshSecret))
    if err != nil { return }
    refresh = tokenPair{Token: rtStr, JTI: jti}

    // Persist hashed refresh token
    rec := models.RefreshToken{
        TokenID:   jti,
        UserIDRef: user.ID,
        TokenHash: utils.SHA256Hex(rtStr),
        ExpiresAt: now.Add(a.RefreshTTL),
    }
    if err = a.DB.Create(&rec).Error; err != nil { return }
    return
}

type refreshRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

func (a *AuthController) Refresh(c *gin.Context) {
    var req refreshRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Parse refresh token
    tok, err := jwt.ParseWithClaims(req.RefreshToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(a.RefreshSecret), nil
    })
    if err != nil || !tok.Valid {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
        return
    }
    claims := tok.Claims.(*jwt.RegisteredClaims)
    // Check DB
    var rec models.RefreshToken
    if err := a.DB.Where("token_hash = ?", utils.SHA256Hex(req.RefreshToken)).First(&rec).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found"})
        return
    }
    if rec.RevokedAt != nil || time.Now().UTC().After(rec.ExpiresAt) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired or revoked"})
        return
    }
    var user models.User
    if err := a.DB.First(&user, rec.UserIDRef).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
        return
    }
    // Rotate refresh token
    access, newRefresh, err := a.issueTokens(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    // Revoke old
    now := time.Now().UTC()
    a.DB.Model(&rec).Updates(map[string]interface{}{
        "revoked_at":         &now,
        "replaced_by_token_id": newRefresh.JTI,
    })
    c.JSON(http.StatusOK, gin.H{
        "access_token": access.Token,
        "token_type":   "Bearer",
        "expires_in":   int(a.AccessTTL.Seconds()),
        "refresh_token": newRefresh.Token,
        "refresh_expires_in": int(a.RefreshTTL.Seconds()),
    })
}

type logoutRequest struct {
    RefreshToken string `json:"refresh_token"`
    All          bool   `json:"all"`
}

// Logout endpoint now revokes refresh tokens (specific or all). Access token remains valid until expiry.
func (a *AuthController) Logout(c *gin.Context) {
    var req logoutRequest
    _ = c.ShouldBindJSON(&req)
    // If specific refresh token provided, revoke it
    if req.RefreshToken != "" {
        var rec models.RefreshToken
        if err := a.DB.Where("token_hash = ?", utils.SHA256Hex(req.RefreshToken)).First(&rec).Error; err == nil {
            now := time.Now().UTC()
            a.DB.Model(&rec).Update("revoked_at", &now)
        }
    }
    if req.All {
        // Revoke all refresh tokens for current user
        if uVal, ok := c.Get("user"); ok {
            user := uVal.(models.User)
            now := time.Now().UTC()
            a.DB.Model(&models.RefreshToken{}).Where("user_id_ref = ? AND revoked_at IS NULL", user.ID).Update("revoked_at", &now)
        }
    }
    c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Logout endpoint for stateless JWT: client should discard token
func (a *AuthController) Logout(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
