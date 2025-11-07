package controllers

import (
    "errors"
    "fmt"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgconn"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type MajorController struct {
    DB *gorm.DB
}

type createMajorRequest struct {
    Code string `json:"code" binding:"required"`
    Name string `json:"name" binding:"required"`
}

type updateMajorRequest struct {
    Code *string `json:"code"`
    Name *string `json:"name"`
}

func (mc *MajorController) ListMajors(c *gin.Context) {
    // Pagination/sort/filter: limit, page, all, sort_by, sort_dir, q
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
        "code":       "code",
        "name":       "name",
    }
    sortCol, ok := allowedSorts[sortBy]
    if !ok {
        sortCol = "created_at"
    }
    order := fmt.Sprintf("%s %s", sortCol, sortDir)

    // Filters
    qText := strings.TrimSpace(c.Query("q"))

    base := mc.DB.Model(&models.Major{})
    if qText != "" {
        like := "%" + qText + "%"
        base = base.Where("code ILIKE ? OR name ILIKE ?", like, like)
    }

    var total int64
    if err := base.Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var majors []models.Major
    listQ := mc.DB.Order(order)
    if qText != "" {
        like := "%" + qText + "%"
        listQ = listQ.Where("code ILIKE ? OR name ILIKE ?", like, like)
    }
    if !all {
        offset := (page - 1) * limit
        listQ = listQ.Offset(offset).Limit(limit)
    }
    if err := listQ.Find(&majors).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    out := make([]gin.H, 0, len(majors))
    for _, m := range majors {
        out = append(out, gin.H{
            "id":         m.ID,
            "code":       m.Code,
            "name":       m.Name,
            "created_at": m.CreatedAt,
            "updated_at": m.UpdatedAt,
        })
    }
    meta := gin.H{"total": total, "all": all}
    if !all {
        meta["limit"] = limit
        meta["page"] = page
        meta["sort_by"] = sortCol
        meta["sort_dir"] = sortDir
    }
    if qText != "" {
        meta["q"] = qText
    }

    c.JSON(http.StatusOK, gin.H{"data": out, "meta": meta})
}

func (mc *MajorController) CreateMajor(c *gin.Context) {
    var req createMajorRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    m := models.Major{Code: req.Code, Name: req.Name}
    if err := mc.DB.Create(&m).Error; err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(http.StatusConflict, gin.H{"error": "major code already exists"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"message": "created", "id": m.ID})
}

func (mc *MajorController) GetMajor(c *gin.Context) {
    id := strings.TrimSpace(c.Param("id"))
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    var m models.Major
    if err := mc.DB.Where("id = ?", id).First(&m).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "major not found"})
        return
    }
    c.JSON(http.StatusOK, m)
}

func (mc *MajorController) UpdateMajor(c *gin.Context) {
    id := strings.TrimSpace(c.Param("id"))
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    var m models.Major
    if err := mc.DB.Where("id = ?", id).First(&m).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "major not found"})
        return
    }

    var req updateMajorRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if req.Code != nil {
        m.Code = *req.Code
    }
    if req.Name != nil {
        m.Name = *req.Name
    }
    if err := mc.DB.Save(&m).Error; err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(http.StatusConflict, gin.H{"error": "major code already exists"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (mc *MajorController) DeleteMajor(c *gin.Context) {
    id := strings.TrimSpace(c.Param("id"))
    if id == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    if err := mc.DB.Where("id = ?", id).Delete(&models.Major{}).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
