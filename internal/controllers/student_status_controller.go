package controllers

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
)

type StudentStatusController struct {
    DB *gorm.DB
}

type updateStatusRequest struct {
    AppVersion string `json:"app_version"`
    Locked     *bool  `json:"locked"`
}

// UpdateSelf allows a siswa to update their app version and lock status.
func (sc *StudentStatusController) UpdateSelf(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)
    if strings.ToLower(user.Role) != "siswa" {
        c.JSON(http.StatusForbidden, gin.H{"error": "only siswa can update status"})
        return
    }

    var req updateStatusRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var st models.StudentStatus
    err := sc.DB.Where("user_id_ref = ?", user.ID).First(&st).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            st = models.StudentStatus{UserIDRef: user.ID}
            if req.AppVersion != "" { st.AppVersion = req.AppVersion }
            if req.Locked != nil { st.Locked = *req.Locked }
            // If locking requested but blocked, prevent
            if st.Locked {
                // Check blocked flag default false
            }
            if err := sc.DB.Create(&st).Error; err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    } else {
        // Existing status
        if req.AppVersion != "" {
            st.AppVersion = req.AppVersion
        }
        if req.Locked != nil {
            // If trying to lock while blocked, deny
            if *req.Locked && st.BlockedFromExam {
                c.JSON(http.StatusForbidden, gin.H{"error": "blocked_by_supervisor"})
                return
            }
            st.Locked = *req.Locked
        }
        if err := sc.DB.Save(&st).Error; err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "app_version":       st.AppVersion,
        "locked":            st.Locked,
        "blocked_from_exam": st.BlockedFromExam,
        "force_logout_at":   st.ForceLogoutAt,
        "updated_at":        st.UpdatedAt,
    })
}

// GetSelf returns current student's status for the app to render UI.
func (sc *StudentStatusController) GetSelf(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)
    if strings.ToLower(user.Role) != "siswa" {
        c.JSON(http.StatusForbidden, gin.H{"error": "only siswa can read status"})
        return
    }
    var st models.StudentStatus
    if err := sc.DB.Where("user_id_ref = ?", user.ID).First(&st).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusOK, gin.H{
                "app_version":       "",
                "locked":            false,
                "blocked_from_exam": false,
                "force_logout_at":   nil,
            })
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "app_version":       st.AppVersion,
        "locked":            st.Locked,
        "blocked_from_exam": st.BlockedFromExam,
        "force_logout_at":   st.ForceLogoutAt,
        "updated_at":        st.UpdatedAt,
    })
}

