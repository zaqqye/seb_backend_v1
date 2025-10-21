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
    JWTSecret string
    ExpiresIn time.Duration
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

    now := time.Now().UTC()
    claims := middleware.Claims{
        UserID: user.UserID,
        Role:   user.Role,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    "seb_backend_v1",
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(a.ExpiresIn)),
            Subject:   strconv.FormatUint(uint64(user.ID), 10),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenStr, err := token.SignedString([]byte(a.JWTSecret))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "access_token": tokenStr,
        "token_type":   "Bearer",
        "expires_in":   int(a.ExpiresIn.Seconds()),
        "role":         user.Role,
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

// Logout endpoint for stateless JWT: client should discard token
func (a *AuthController) Logout(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
