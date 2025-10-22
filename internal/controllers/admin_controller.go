package controllers

import (
    "fmt"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/utils"
)

type AdminController struct {
    DB *gorm.DB
}

func (a *AdminController) ListUsers(c *gin.Context) {
    // Query params: limit, page, all, sort_by, sort_dir, q, role, active
    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 20
    page := 1
    if v := c.Query("limit"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 {
            limit = n
        }
    }
    if v := c.Query("page"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 {
            page = n
        }
    }

    sortBy := strings.ToLower(c.DefaultQuery("sort_by", "created_at"))
    sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "DESC"))
    if sortDir != "ASC" && sortDir != "DESC" {
        sortDir = "DESC"
    }
    allowedSorts := map[string]string{
        "id":         "id",
        "created_at": "created_at",
        "full_name":  "full_name",
        "email":      "email",
        "role":       "role",
        "kelas":      "kelas",
        "jurusan":    "jurusan",
        "active":     "active",
    }
    sortCol, ok := allowedSorts[sortBy]
    if !ok {
        sortCol = "created_at"
    }
    order := fmt.Sprintf("%s %s", sortCol, sortDir)
    
    // Filters
    qText := strings.TrimSpace(c.Query("q"))
    role := strings.TrimSpace(strings.ToLower(c.Query("role")))
    activeStr := strings.TrimSpace(strings.ToLower(c.Query("active")))

    base := a.DB.Model(&models.User{})
    if qText != "" {
        like := "%" + qText + "%"
        base = base.Where("full_name ILIKE ? OR email ILIKE ?", like, like)
    }
    if role != "" {
        if !IsValidRole(role) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
            return
        }
        base = base.Where("role = ?", role)
    }
    if activeStr != "" {
        switch activeStr {
        case "true", "1":
            base = base.Where("active = ?", true)
        case "false", "0":
            base = base.Where("active = ?", false)
        default:
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid active value"})
            return
        }
    }

    var total int64
    if err := base.Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var users []models.User
    listQ := a.DB.Order(order)
    // reapply filters to list query
    if qText != "" {
        like := "%" + qText + "%"
        listQ = listQ.Where("full_name ILIKE ? OR email ILIKE ?", like, like)
    }
    if role != "" {
        listQ = listQ.Where("role = ?", role)
    }
    if activeStr != "" {
        switch activeStr {
        case "true", "1":
            listQ = listQ.Where("active = ?", true)
        case "false", "0":
            listQ = listQ.Where("active = ?", false)
        }
    }
    if !all {
        offset := (page - 1) * limit
        listQ = listQ.Offset(offset).Limit(limit)
    }
    if err := listQ.Find(&users).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    out := make([]gin.H, 0, len(users))
    for _, u := range users {
        out = append(out, gin.H{
            "id":         u.ID,
            "user_id":    u.UserID,
            "full_name":  u.FullName,
            "email":      u.Email,
            "role":       u.Role,
            "kelas":      u.Kelas,
            "jurusan":    u.Jurusan,
            "active":     u.Active,
            "created_at": u.CreatedAt,
            "updated_at": u.UpdatedAt,
        })
    }
    meta := gin.H{"total": total, "all": all}
    if !all {
        meta["limit"] = limit
        meta["page"] = page
        meta["sort_by"] = sortCol
        meta["sort_dir"] = sortDir
    }
    // reflect filters in meta
    if qText != "" {
        meta["q"] = qText
    }
    if role != "" {
        meta["role"] = role
    }
    if activeStr != "" {
        meta["active"] = activeStr
    }
    c.JSON(http.StatusOK, gin.H{"data": out, "meta": meta})
}

func (a *AdminController) GetUser(c *gin.Context) {
    userID := c.Param("user_id")
    var u models.User
    if err := a.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "id":         u.ID,
        "user_id":    u.UserID,
        "full_name":  u.FullName,
        "email":      u.Email,
        "role":       u.Role,
        "kelas":      u.Kelas,
        "jurusan":    u.Jurusan,
        "active":     u.Active,
        "created_at": u.CreatedAt,
        "updated_at": u.UpdatedAt,
    })
}

type updateUserRequest struct {
    FullName *string `json:"full_name"`
    Email    *string `json:"email"`
    Password *string `json:"password"`
    Role     *string `json:"role"`
    Kelas    *string `json:"kelas"`
    Jurusan  *string `json:"jurusan"`
    Active   *bool   `json:"active"`
}

func (a *AdminController) UpdateUser(c *gin.Context) {
    userID := c.Param("user_id")
    var u models.User
    if err := a.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    var req updateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.FullName != nil {
        u.FullName = *req.FullName
    }
    if req.Email != nil {
        u.Email = *req.Email
    }
    if req.Role != nil {
        if !IsValidRole(*req.Role) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
            return
        }
        u.Role = *req.Role
    }
    if req.Kelas != nil {
        u.Kelas = *req.Kelas
    }
    if req.Jurusan != nil {
        u.Jurusan = *req.Jurusan
    }
    if req.Active != nil {
        u.Active = *req.Active
    }
    if req.Password != nil && *req.Password != "" {
        pw, err := utils.HashPassword(*req.Password)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
            return
        }
        u.Password = pw
    }

    if err := a.DB.Save(&u).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (a *AdminController) DeleteUser(c *gin.Context) {
    userID := c.Param("user_id")
    if err := a.DB.Where("user_id = ?", userID).Delete(&models.User{}).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
