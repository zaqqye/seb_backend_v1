package controllers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/config"
    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type ConfigController struct {
    DB  *gorm.DB
    Cfg *config.Config
}

func (cc *ConfigController) Get(c *gin.Context) {
    platform := c.DefaultQuery("platform", "android")
    var role string
    if u, ok := c.Get("user"); ok {
        role = u.(models.User).Role
    }

    flags := gin.H{
        "showExitCode":         true,
        "newDashboardLayoutV2": false,
        "enableOfflineMode":    false,
    }
    if role == "admin" {
        flags["newDashboardLayoutV2"] = true
    }

    minApp := cc.Cfg.MinAppVersionAndroid
    if platform == "ios" || platform == "ipad" || platform == "iphone" {
        minApp = cc.Cfg.MinAppVersionIOS
    }
    minAppInt, _ := strconv.Atoi(minApp)

    c.JSON(http.StatusOK, gin.H{
        "platform":        platform,
        "layout_version":  cc.Cfg.LayoutVersion,
        "min_app_version": minAppInt,
        "flags":           flags,
        "schema_version":  1,
    })
}

