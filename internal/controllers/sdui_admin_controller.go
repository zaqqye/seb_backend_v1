package controllers

import (
    "encoding/json"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type SDUIAdminController struct {
    DB *gorm.DB
}

type sduiCreateRequest struct {
    Name          string      `json:"name" binding:"required"`
    Platform      string      `json:"platform" binding:"required"`
    Role          string      `json:"role"`
    SchemaVersion int         `json:"schema_version" binding:"required"`
    ScreenVersion int         `json:"screen_version" binding:"required"`
    Active        *bool       `json:"active"`
    Payload       interface{} `json:"payload" binding:"required"`
}

func (a *SDUIAdminController) Create(c *gin.Context) {
    var req sduiCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if req.Active == nil { b := true; req.Active = &b }
    raw, err := json.Marshal(req.Payload)
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"}); return }
    rec := models.SduiScreen{
        Name:          strings.ToLower(req.Name),
        Platform:      strings.ToLower(req.Platform),
        Role:          strings.ToLower(req.Role),
        SchemaVersion: req.SchemaVersion,
        ScreenVersion: req.ScreenVersion,
        Active:        *req.Active,
        Payload:       raw,
    }
    if err := a.DB.Create(&rec).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"id": rec.ID})
}

type sduiUpdateRequest struct {
    Name          *string     `json:"name"`
    Platform      *string     `json:"platform"`
    Role          *string     `json:"role"`
    SchemaVersion *int        `json:"schema_version"`
    ScreenVersion *int        `json:"screen_version"`
    Active        *bool       `json:"active"`
    Payload       interface{} `json:"payload"`
}

func (a *SDUIAdminController) Update(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    var rec models.SduiScreen
    if err := a.DB.First(&rec, id).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }

    var req sduiUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    if req.Name != nil { rec.Name = strings.ToLower(*req.Name) }
    if req.Platform != nil { rec.Platform = strings.ToLower(*req.Platform) }
    if req.Role != nil { rec.Role = strings.ToLower(*req.Role) }
    if req.SchemaVersion != nil { rec.SchemaVersion = *req.SchemaVersion }
    if req.ScreenVersion != nil { rec.ScreenVersion = *req.ScreenVersion }
    if req.Active != nil { rec.Active = *req.Active }
    if req.Payload != nil {
        raw, err := json.Marshal(req.Payload)
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"}); return }
        rec.Payload = raw
    }
    if err := a.DB.Save(&rec).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (a *SDUIAdminController) Get(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    var rec models.SduiScreen
    if err := a.DB.First(&rec, id).Error; err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }
    c.JSON(http.StatusOK, rec)
}

func (a *SDUIAdminController) Delete(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    if err := a.DB.Delete(&models.SduiScreen{}, id).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (a *SDUIAdminController) List(c *gin.Context) {
    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 20
    page := 1
    if v := c.Query("limit"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { limit = n } }
    if v := c.Query("page"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { page = n } }
    sortBy := strings.ToLower(c.DefaultQuery("sort_by", "updated_at"))
    sortDir := strings.ToUpper(c.DefaultQuery("sort_dir", "DESC"))
    if sortDir != "ASC" && sortDir != "DESC" { sortDir = "DESC" }
    allowedSorts := map[string]string{"updated_at": "updated_at", "name": "name", "platform": "platform", "screen_version": "screen_version"}
    sortCol, ok := allowedSorts[sortBy]; if !ok { sortCol = "updated_at" }
    order := sortCol + " " + sortDir

    qname := strings.ToLower(strings.TrimSpace(c.Query("name")))
    qplat := strings.ToLower(strings.TrimSpace(c.Query("platform")))
    qrole := strings.ToLower(strings.TrimSpace(c.Query("role")))
    qactive := strings.ToLower(strings.TrimSpace(c.Query("active")))

    base := a.DB.Model(&models.SduiScreen{})
    if qname != "" { base = base.Where("name = ?", qname) }
    if qplat != "" { base = base.Where("platform = ?", qplat) }
    if qrole != "" { base = base.Where("role = ?", qrole) }
    if qactive != "" {
        switch qactive { case "true","1": base = base.Where("active = ?", true); case "false","0": base = base.Where("active = ?", false); }
    }

    var total int64
    if err := base.Count(&total).Error; err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }

    listQ := a.DB.Model(&models.SduiScreen{}).Order(order)
    if qname != "" { listQ = listQ.Where("name = ?", qname) }
    if qplat != "" { listQ = listQ.Where("platform = ?", qplat) }
    if qrole != "" { listQ = listQ.Where("role = ?", qrole) }
    if qactive != "" {
        switch qactive { case "true","1": listQ = listQ.Where("active = ?", true); case "false","0": listQ = listQ.Where("active = ?", false) }
    }
    if !all { listQ = listQ.Offset((page-1)*limit).Limit(limit) }

    var items []models.SduiScreen
    if err := listQ.Find(&items).Error; err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }

    meta := gin.H{"total": total, "all": all}
    if !all { meta["limit"] = limit; meta["page"] = page; meta["sort_by"] = sortBy; meta["sort_dir"] = sortDir }
    c.JSON(http.StatusOK, gin.H{"data": items, "meta": meta})
}

