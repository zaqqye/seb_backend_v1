package controllers

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/ws"
)

type StudentStatusController struct {
    DB   *gorm.DB
    Hubs *ws.Hubs
}

type updateStatusRequest struct {
    AppVersion      string `json:"app_version"`
    Locked          *bool  `json:"locked"`
    BlockedFromExam *bool  `json:"blocked_from_exam"`
}

// UpdateSelf allows a siswa to update their app version and lock status.
func (sc *StudentStatusController) UpdateSelf(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)
    role := strings.ToLower(user.Role)
    if role != "siswa" && role != "admin" && role != "pengawas" {
        c.JSON(http.StatusForbidden, gin.H{"error": "role_not_allowed"})
        return
    }

    var req updateStatusRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.BlockedFromExam != nil && !*req.BlockedFromExam && role == "siswa" {
        c.JSON(http.StatusForbidden, gin.H{"error": "blocked_from_exam_cannot_be_cleared_by_siswa"})
        return
    }

    var st models.StudentStatus
    err := sc.DB.Where("user_id_ref = ?", user.ID).First(&st).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            st = models.StudentStatus{UserIDRef: user.ID}
            if req.AppVersion != "" { st.AppVersion = req.AppVersion }
            if req.Locked != nil { st.Locked = *req.Locked }
            if req.BlockedFromExam != nil { st.BlockedFromExam = *req.BlockedFromExam }
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
        if req.BlockedFromExam != nil {
            st.BlockedFromExam = *req.BlockedFromExam
        }
        if err := sc.DB.Save(&st).Error; err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    }
    go broadcastStudentStatus(sc.DB, sc.Hubs, user.ID)

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
    role := strings.ToLower(user.Role)
    if role != "siswa" && role != "admin" && role != "pengawas" {
        c.JSON(http.StatusForbidden, gin.H{"error": "role_not_allowed"})
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
