package controllers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgconn"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"

    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/utils"
    "github.com/zaqqye/seb_backend_v1/internal/ws"
)

type ExitCodeController struct {
    DB   *gorm.DB
    Hubs *ws.Hubs
}

var errNotAllowedForRoom = errors.New("not allowed for this room")

// Helper: return allowed room IDs for a user. Admins: nil means all.
func (ec *ExitCodeController) allowedRoomIDsFor(user models.User) ([]string, bool, error) {
    if user.Role == "admin" {
        return nil, true, nil // all rooms allowed
    }
    if user.Role == "pengawas" {
        var m []models.RoomSupervisor
        if err := ec.DB.Where("user_id_ref = ?", user.ID).Find(&m).Error; err != nil {
            return nil, false, err
        }
        ids := make([]string, 0, len(m))
        for _, r := range m {
            ids = append(ids, r.RoomIDRef)
        }
        return ids, false, nil
    }
    return []string{}, false, nil // siswa: no access
}

type generateExitCodeRequest struct {
    RoomID      *string  `json:"room_id"`        // required: determines which room's students are targeted
    Length      int    `json:"length"`         // optional length of generated code; defaults to 6
    StudentIDs  []string `json:"student_ids"`    // optional list of specific students within the room
    AllStudents bool   `json:"all_students"`   // when true, generate codes for every student in the room
    SingleForRoom bool `json:"single_for_room"` // when true, generate one reusable code for the room
}

func (ec *ExitCodeController) Generate(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    var req generateExitCodeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.RoomID == nil || strings.TrimSpace(*req.RoomID) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "room_id is required"})
        return
    }
    trimmedRoomID := strings.TrimSpace(*req.RoomID)
    req.RoomID = &trimmedRoomID
    if req.SingleForRoom {
        // room-wide code does not require student_ids/all_students
        // continue
    } else {
        if !req.AllStudents && len(req.StudentIDs) == 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "student_ids is required unless all_students is true"})
            return
        }
        if req.AllStudents && len(req.StudentIDs) > 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "student_ids must be empty when all_students is true"})
            return
        }
    }

    allowedRooms, isAdmin, err := ec.allowedRoomIDsFor(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Validate room permission for pengawas
    if !isAdmin {
        permitted := false
        for _, rid := range allowedRooms {
            if rid == *req.RoomID {
                permitted = true
                break
            }
        }
        if !permitted {
            c.JSON(http.StatusForbidden, gin.H{"error": "not allowed for this room"})
            return
        }
    }

    // Ensure room exists
    var room models.Room
    if err := ec.DB.Where("id = ?", *req.RoomID).First(&room).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
        return
    }

    var roomStudents []models.RoomStudent
    if err := ec.DB.Where("room_id_ref = ?", *req.RoomID).Find(&roomStudents).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    studentInRoom := make(map[string]struct{}, len(roomStudents))
    for _, rs := range roomStudents {
        studentInRoom[rs.UserIDRef] = struct{}{}
    }

    var targetStudentIDs []string
    if req.AllStudents {
        for _, rs := range roomStudents {
            targetStudentIDs = append(targetStudentIDs, rs.UserIDRef)
        }
    } else {
        seen := make(map[string]struct{}, len(req.StudentIDs))
        for _, sid := range req.StudentIDs {
            sid = strings.TrimSpace(sid)
            if sid == "" {
                c.JSON(http.StatusBadRequest, gin.H{"error": "student_ids cannot contain blank values"})
                return
            }
            if _, ok := seen[sid]; ok {
                continue
            }
            if _, ok := studentInRoom[sid]; !ok {
                c.JSON(http.StatusBadRequest, gin.H{"error": "student is not assigned to the specified room"})
                return
            }
            seen[sid] = struct{}{}
            targetStudentIDs = append(targetStudentIDs, sid)
        }
    }

    if !req.SingleForRoom {
        if len(targetStudentIDs) == 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "no students found for code generation"})
            return
        }
    }

    studentUUIDs, err := toUUIDSlice(targetStudentIDs)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Sanity check: ensure all target users exist and have siswa role.
    if !req.SingleForRoom {
        var students []models.User
        if err := ec.DB.Where("id IN ?", studentUUIDs).Find(&students).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        if len(students) != len(targetStudentIDs) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "one or more students are invalid"})
            return
        }
        for _, s := range students {
            if strings.ToLower(s.Role) != "siswa" {
                c.JSON(http.StatusBadRequest, gin.H{"error": "exit code can only be generated for siswa users"})
                return
            }
        }
    }

    created := make([]models.ExitCode, 0, 1)
    err = ec.DB.Transaction(func(tx *gorm.DB) error {
        if req.SingleForRoom {
            const maxAttempts = 5
            var rec models.ExitCode
            for attempt := 0; attempt < maxAttempts; attempt++ {
                code, genErr := utils.GenerateCode(req.Length)
                if genErr != nil { return genErr }
                rec = models.ExitCode{
                    UserIDRef:  user.ID,
                    RoomIDRef:  req.RoomID,
                    Code:       code,
                    Reusable:   true,
                }
                if err := tx.Create(&rec).Error; err != nil {
                    var pgErr *pgconn.PgError
                    if errors.As(err, &pgErr) && pgErr.Code == "23505" && attempt < maxAttempts-1 { continue }
                    return err
                }
                break
            }
            if rec.ID == "" { return errors.New("failed to generate exit code") }
            created = append(created, rec)
            return nil
        }
        for _, studentID := range targetStudentIDs {
            // Mark previous unused codes for this student (same room context) as used
            now := time.Now().UTC()
            markQ := tx.Model(&models.ExitCode{}).
                Where("student_user_id_ref = ? AND used_at IS NULL", studentID)
            if req.RoomID != nil {
                markQ = markQ.Where("room_id_ref = ?", *req.RoomID)
            } else {
                markQ = markQ.Where("room_id_ref IS NULL")
            }
            if err := markQ.Update("used_at", now).Error; err != nil {
                return err
            }

            const maxAttempts = 5
            var rec models.ExitCode
            for attempt := 0; attempt < maxAttempts; attempt++ {
                code, genErr := utils.GenerateCode(req.Length)
                if genErr != nil {
                    return genErr
                }
                sid := studentID
                rec = models.ExitCode{
                    UserIDRef:        user.ID,
                    StudentUserIDRef: &sid,
                    RoomIDRef:        req.RoomID,
                    Code:             code,
                }
                if err := tx.Create(&rec).Error; err != nil {
                    var pgErr *pgconn.PgError
                    if errors.As(err, &pgErr) && pgErr.Code == "23505" && attempt < maxAttempts-1 {
                        continue
                    }
                    return err
                }
                break
            }
            if rec.ID == "" {
                return errors.New("failed to generate exit code")
            }
            created = append(created, rec)
        }
        return nil
    })
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(http.StatusConflict, gin.H{"error": "code already exists, retry"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    out := make([]gin.H, 0, len(created))
    for _, rec := range created {
        item := gin.H{
            "id":               rec.ID,
            "code":             rec.Code,
            "student_user_id":  rec.StudentUserIDRef,
            "created_at":       rec.CreatedAt,
            "created_by":       rec.UserIDRef,
        }
        if rec.RoomIDRef != nil {
            item["room_id"] = *rec.RoomIDRef
        }
        out = append(out, item)
    }

    c.JSON(http.StatusCreated, gin.H{"data": out})
}

func (ec *ExitCodeController) List(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    // Query params: limit, page, all, sort_by, sort_dir, room_id, used
    all := strings.EqualFold(c.Query("all"), "true") || c.Query("all") == "1"
    limit := 50
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
        "id":               "ec.id",
        "created_at":       "ec.created_at",
        "used_at":          "ec.used_at",
        "code":             "ec.code",
        "student_user_id":  "ec.student_user_id_ref",
    }
    sortCol, ok := allowedSorts[sortBy]
    if !ok {
        sortCol = "created_at"
    }
    order := sortCol + " " + sortDir

    allowedRooms, isAdmin, err := ec.allowedRoomIDsFor(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if !isAdmin && len(allowedRooms) == 0 {
        c.JSON(http.StatusOK, gin.H{"data": []any{}, "meta": gin.H{"total": 0, "all": all}})
        return
    }

    roomFilter := strings.TrimSpace(c.Query("room_id"))
    studentFilter := strings.TrimSpace(c.Query("student_user_id"))
    usedFilter := strings.TrimSpace(strings.ToLower(c.DefaultQuery("used", "false")))

    applyFilters := func(q *gorm.DB, alias string) *gorm.DB {
        col := func(field string) string {
            if alias != "" {
                return alias + "." + field
            }
            return field
        }
        if !isAdmin {
            if len(allowedRooms) == 0 {
                return q.Where("1=0")
            }
            // Use supervisor join to preserve index usage
            if alias == "ec" {
                q = q.Joins("JOIN room_supervisors sup ON sup.room_id_ref = ec.room_id_ref AND sup.user_id_ref = ?", user.ID)
            } else {
                q = q.Joins("JOIN room_supervisors sup ON sup.room_id_ref = "+col("room_id_ref")+" AND sup.user_id_ref = ?", user.ID)
            }
        }
        if roomFilter != "" {
            q = q.Where(col("room_id_ref")+" = ?", roomFilter)
        }
        if studentFilter != "" {
            q = q.Where(col("student_user_id_ref")+" = ?", studentFilter)
        }
        switch usedFilter {
        case "true", "1":
            q = q.Where(col("used_at") + " IS NOT NULL")
        case "all":
            // no filter
        case "false", "0":
            fallthrough
        default:
            q = q.Where(col("used_at") + " IS NULL")
        }
        return q
    }

    base := applyFilters(ec.DB.Model(&models.ExitCode{}), "")

    var total int64
    if err := base.Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    type exitCodeRow struct {
        models.ExitCode
        RoomName    string `gorm:"column:room_name"`
        StudentName string `gorm:"column:student_name"`
    }

    listQ := applyFilters(
        ec.DB.Table("exit_codes AS ec").
            Select("ec.*, COALESCE(u.full_name, '') AS student_name, COALESCE(r.name, '') AS room_name").
            Joins("LEFT JOIN users u ON u.id = ec.student_user_id_ref").
            Joins("LEFT JOIN rooms r ON r.id = ec.room_id_ref"),
        "ec",
    ).Order(order)
    if !all {
        offset := (page - 1) * limit
        listQ = listQ.Offset(offset).Limit(limit)
    }
    var items []exitCodeRow
    if err := listQ.Find(&items).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    out := make([]gin.H, 0, len(items))
    for _, e := range items {
        status := "used"
        if e.UsedAt == nil {
            status = "unused"
        }
        out = append(out, gin.H{
            "id":              e.ID,
            "room_id":         e.RoomIDRef,
            "room_name":       e.RoomName,
            "student_user_id": e.StudentUserIDRef,
            "student_name":    e.StudentName,
            "code":            e.Code,
            "used_at":         e.UsedAt,
            "status":          status,
            "created_at":      e.CreatedAt,
            "created_by":      e.UserIDRef,
        })
    }
    meta := gin.H{"total": total, "all": all}
    if !all {
        meta["limit"] = limit
        meta["page"] = page
        meta["sort_by"] = sortBy
        meta["sort_dir"] = sortDir
    }
    c.JSON(http.StatusOK, gin.H{"data": out, "meta": meta})
}

func (ec *ExitCodeController) Revoke(c *gin.Context) {
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    idStr := strings.TrimSpace(c.Param("id"))
    if idStr == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    var rec models.ExitCode
    if err := ec.DB.Where("id = ?", idStr).First(&rec).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "exit code not found"})
        return
    }

    // Scope check for pengawas
    if user.Role == "pengawas" {
        var count int64
        if rec.RoomIDRef == nil {
            // Pengawas cannot revoke non-room-specific codes
            c.JSON(http.StatusForbidden, gin.H{"error": "not allowed to revoke this code"})
            return
        }
        if err := ec.DB.Model(&models.RoomSupervisor{}).
            Where("user_id_ref = ? AND room_id_ref = ?", user.ID, *rec.RoomIDRef).
            Count(&count).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        if count == 0 {
            c.JSON(http.StatusForbidden, gin.H{"error": "not allowed for this room"})
            return
        }
    }

    if rec.UsedAt == nil {
        now := time.Now().UTC()
        rec.UsedAt = &now
        if err := ec.DB.Save(&rec).Error; err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
    }
    c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

// Consume marks a code as used (single-use). If already used, returns 409.
type consumeRequest struct {
    Code          string  `json:"code" binding:"required"`
    RoomID        *string `json:"room_id"`
    StudentUserID *string `json:"student_user_id"`
}

func (ec *ExitCodeController) Consume(c *gin.Context) {
    // Optional: if siswa memanggil endpoint ini, setelah berhasil konsumsi kode,
    // status akan di-set ke locked=false (keluar dari mode terkunci)
    uVal, _ := c.Get("user")
    user := uVal.(models.User)

    var req consumeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.RoomID != nil {
        trimmed := strings.TrimSpace(*req.RoomID)
        if trimmed == "" {
            req.RoomID = nil
        } else {
            req.RoomID = &trimmed
        }
    }

    role := strings.ToLower(user.Role)
    var targetStudentID string
    switch role {
    case "siswa":
        targetStudentID = user.ID
        if req.StudentUserID != nil && strings.TrimSpace(*req.StudentUserID) != "" && strings.TrimSpace(*req.StudentUserID) != targetStudentID {
            c.JSON(http.StatusForbidden, gin.H{"error": "cannot consume code for another student"})
            return
        }
    default:
        if req.StudentUserID == nil || strings.TrimSpace(*req.StudentUserID) == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "student_user_id is required"})
            return
        }
        targetStudentID = strings.TrimSpace(*req.StudentUserID)
    }

    now := time.Now().UTC()

    allowedRooms, isAdmin, err := ec.allowedRoomIDsFor(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var consumed models.ExitCode
    err = ec.DB.Transaction(func(tx *gorm.DB) error {
        // Try student-specific code first
        q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Where("code = ? AND used_at IS NULL AND student_user_id_ref = ?", req.Code, targetStudentID)
        if req.RoomID != nil {
            q = q.Where("room_id_ref = ?", *req.RoomID)
        }
        err := q.First(&consumed).Error
        if err != nil {
            if !errors.Is(err, gorm.ErrRecordNotFound) {
                return err
            }
            // Fallback: room-wide reusable code
            rq := tx.Where("code = ? AND reusable = ?", req.Code, true)
            if req.RoomID != nil {
                rq = rq.Where("room_id_ref = ?", *req.RoomID)
            }
            if err := rq.First(&consumed).Error; err != nil {
                return err
            }
            // Validate student belongs to the code room
            if consumed.RoomIDRef == nil {
                return gorm.ErrRecordNotFound
            }
            var count int64
            if err := tx.Model(&models.RoomStudent{}).Where("user_id_ref = ? AND room_id_ref = ?", targetStudentID, *consumed.RoomIDRef).Count(&count).Error; err != nil {
                return err
            }
            if count == 0 {
                return errNotAllowedForRoom
            }
            // Validate pengawas scope if caller pengawas
            if role == "pengawas" {
                permitted := false
                for _, rid := range allowedRooms {
                    if rid == *consumed.RoomIDRef { permitted = true; break }
                }
                if !permitted { return errNotAllowedForRoom }
            }
            // Reusable code: do not mark used_at, allow multiple students
            return nil
        }
        if role == "pengawas" {
            if consumed.RoomIDRef == nil { return errNotAllowedForRoom }
            permitted := false
            for _, rid := range allowedRooms { if rid == *consumed.RoomIDRef { permitted = true; break } }
            if !permitted { return errNotAllowedForRoom }
        }
        if !isAdmin && role != "pengawas" && role != "siswa" { return errNotAllowedForRoom }
        consumed.UsedAt = &now
        if err := tx.Save(&consumed).Error; err != nil { return err }
        return nil
    })
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusConflict, gin.H{"error": "code not found or already used"})
            return
        }
        if errors.Is(err, errNotAllowedForRoom) {
            c.JSON(http.StatusForbidden, gin.H{"error": errNotAllowedForRoom.Error()})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Jika pemanggil adalah siswa, set status locked=false
    if role == "siswa" {
        var st models.StudentStatus
        if err := ec.DB.Where("user_id_ref = ?", user.ID).First(&st).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                st = models.StudentStatus{UserIDRef: user.ID, Locked: false}
                _ = ec.DB.Create(&st).Error
            }
        } else {
            st.Locked = false
            _ = ec.DB.Save(&st).Error
        }
    }
    go broadcastStudentStatus(ec.DB, ec.Hubs, targetStudentID)
    c.JSON(http.StatusOK, gin.H{"message": "consumed"})
}
