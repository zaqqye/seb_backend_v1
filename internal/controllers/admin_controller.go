package controllers

import (
    "bytes"
    "encoding/csv"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/zaqqye/seb_backend_v1/internal/models"
    "github.com/zaqqye/seb_backend_v1/internal/utils"
)

type AdminController struct {
    DB *gorm.DB
}

type userImportError struct {
    Row   int    `json:"row"`
    Email string `json:"email,omitempty"`
    Error string `json:"error"`
}

func parseBoolDefaultTrue(val string) (bool, bool) {
    if val == "" {
        return true, false
    }
    switch strings.ToLower(strings.TrimSpace(val)) {
    case "true", "1", "yes", "y", "aktif":
        return true, true
    case "false", "0", "no", "n", "nonaktif", "inactive":
        return false, true
    default:
        return true, false
    }
}

// ImportUsers allows admin to bulk-create users from a CSV file.
// Expected header columns (case-insensitive):
// full_name, email, password, role (optional), kelas (optional), jurusan (optional), active (optional), room_name (optional)
func (a *AdminController) ImportUsers(c *gin.Context) {
    // Limit max upload size (10MB) to avoid accidental huge files.
    if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
        return
    }
    file, fileHeader, err := c.Request.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
        return
    }
    defer file.Close()

    if fileHeader == nil || fileHeader.Filename == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file name"})
        return
    }
    filename := strings.ToLower(strings.TrimSpace(fileHeader.Filename))
    if !strings.HasSuffix(filename, ".csv") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "only .csv files are allowed"})
        return
    }

    data, err := io.ReadAll(file)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
        return
    }
    if len(bytes.TrimSpace(data)) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "file is empty"})
        return
    }

    // Normalise line endings so files saved with only CR (Mac classic) or CRLF behave consistently.
    data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
    data = bytes.ReplaceAll(data, []byte{'\r'}, []byte{'\n'})

    delimiter := ','
    firstLineEnd := bytes.IndexByte(data, '\n')
    if firstLineEnd == -1 {
        firstLineEnd = len(data)
    }
    firstLine := data[:firstLineEnd]
    firstLine = bytes.TrimPrefix(firstLine, []byte{0xEF, 0xBB, 0xBF})
    if bytes.Contains(firstLine, []byte{';'}) && !bytes.Contains(firstLine, []byte{','}) {
        delimiter = ';'
    }

    newReader := func() *csv.Reader {
        r := csv.NewReader(bytes.NewReader(data))
        r.TrimLeadingSpace = true
        r.FieldsPerRecord = -1
        if delimiter != ',' {
            r.Comma = delimiter
        }
        return r
    }

    reader := newReader()
    header, err := reader.Read()
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read header"})
        return
    }
    cleanHeader := func(val string) string {
        v := strings.TrimSpace(val)
        for strings.HasPrefix(v, "\ufeff") {
            v = strings.TrimPrefix(v, "\ufeff")
        }
        v = strings.Trim(v, "\"'")
        return v
    }
    for i := range header {
        header[i] = cleanHeader(header[i])
    }

    headerIdx := make(map[string]int, len(header))
    for idx, col := range header {
        key := strings.ToLower(strings.TrimSpace(col))
        if key != "" {
            headerIdx[key] = idx
        }
    }
    log.Printf("import csv headers: %+v", header)

    required := []string{"full_name", "email", "password"}
    for _, key := range required {
        if _, ok := headerIdx[key]; !ok {
            c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("missing header column: %s", key)})
            return
        }
    }

    getVal := func(record []string, key string) string {
        idx, ok := headerIdx[key]
        if !ok || idx >= len(record) {
            return ""
        }
        return strings.TrimSpace(record[idx])
    }

    var (
        totalRows   int
        createdRows int
        failures    []userImportError
    )

    rowNum := 1 // already consumed header line
    roomCache := make(map[string]models.Room)
    for {
        row, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            failures = append(failures, userImportError{
                Row:   rowNum + 1,
                Error: fmt.Sprintf("failed to read row: %v", err),
            })
            continue
        }
        rowNum++
        totalRows++

        fullName := getVal(row, "full_name")
        email := strings.ToLower(getVal(row, "email"))
        password := getVal(row, "password")
        role := strings.ToLower(getVal(row, "role"))
        kelas := getVal(row, "kelas")
        jurusan := getVal(row, "jurusan")
        activeStr := getVal(row, "active")
        roomName := getVal(row, "room_name")

        if fullName == "" || email == "" || password == "" {
            failures = append(failures, userImportError{
                Row:   rowNum,
                Email: email,
                Error: "full_name, email, and password are required",
            })
            continue
        }

        if role == "" {
            role = "siswa"
        }
        if !IsValidRole(role) {
            failures = append(failures, userImportError{
                Row:   rowNum,
                Email: email,
                Error: "invalid role",
            })
            continue
        }

        activeVal, provided := parseBoolDefaultTrue(activeStr)
        if activeStr != "" && !provided {
            failures = append(failures, userImportError{
                Row:   rowNum,
                Email: email,
                Error: "invalid active value",
            })
            continue
        }

        if existingErr := a.DB.Where("email = ?", email).First(&models.User{}).Error; existingErr == nil {
            failures = append(failures, userImportError{
                Row:   rowNum,
                Email: email,
                Error: "email already exists",
            })
            continue
        } else if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
            failures = append(failures, userImportError{
                Row:   rowNum,
                Email: email,
                Error: fmt.Sprintf("failed to check existing user: %v", existingErr),
            })
            continue
        }

        hashed, hashErr := utils.HashPassword(password)
        if hashErr != nil {
            failures = append(failures, userImportError{
                Row:   rowNum,
                Email: email,
                Error: fmt.Sprintf("failed to hash password: %v", hashErr),
            })
            continue
        }

        user := models.User{
            FullName: fullName,
            Email:    email,
            Password: hashed,
            Role:     role,
            Kelas:    kelas,
            Jurusan:  jurusan,
            Active:   activeVal,
        }

        if err := a.DB.Transaction(func(tx *gorm.DB) error {
            if err := tx.Create(&user).Error; err != nil {
                return err
            }
            if roomName != "" {
                if role != "siswa" {
                    return fmt.Errorf("room assignment only allowed for siswa role")
                }
                normalized := strings.ToLower(roomName)
                room, ok := roomCache[normalized]
                if !ok {
                    var fetched models.Room
                    if err := tx.Where("LOWER(name) = ?", normalized).First(&fetched).Error; err != nil {
                        if errors.Is(err, gorm.ErrRecordNotFound) {
                            return fmt.Errorf("room '%s' not found", roomName)
                        }
                        return err
                    }
                    roomCache[normalized] = fetched
                    room = fetched
                }
                assignment := models.RoomStudent{UserIDRef: user.ID, RoomIDRef: room.ID}
                if err := tx.Where("user_id_ref = ? AND room_id_ref = ?", assignment.UserIDRef, assignment.RoomIDRef).
                    FirstOrCreate(&assignment).Error; err != nil {
                    return err
                }
            }
            return nil
        }); err != nil {
            failures = append(failures, userImportError{
                Row:   rowNum,
                Email: email,
                Error: fmt.Sprintf("failed to insert user: %v", err),
            })
            continue
        }

        createdRows++
    }

    c.JSON(http.StatusOK, gin.H{
        "summary": gin.H{
            "total_rows": totalRows,
            "inserted":   createdRows,
            "failed":     len(failures),
        },
        "errors": failures,
    })
}

func (a *AdminController) ListUsers(c *gin.Context) {
    // Query params: limit, page, all, sort_by, sort_dir, q, role, active
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
    kelasFilter := strings.TrimSpace(strings.ToLower(c.Query("kelas")))
    jurusanFilter := strings.TrimSpace(strings.ToLower(c.Query("jurusan")))

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
    if kelasFilter != "" {
        base = base.Where("LOWER(kelas) = ?", kelasFilter)
    }
    if jurusanFilter != "" {
        base = base.Where("LOWER(jurusan) = ?", jurusanFilter)
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
    if kelasFilter != "" {
        listQ = listQ.Where("LOWER(kelas) = ?", kelasFilter)
    }
    if jurusanFilter != "" {
        listQ = listQ.Where("LOWER(jurusan) = ?", jurusanFilter)
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

    userIDs := make([]string, 0, len(users))
    for _, u := range users {
        userIDs = append(userIDs, u.ID)
    }
    uuidUserIDs := make([]uuid.UUID, 0, len(userIDs))
    for _, idStr := range userIDs {
        parsed, err := uuid.Parse(idStr)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("invalid user id format: %s", err.Error())})
            return
        }
        uuidUserIDs = append(uuidUserIDs, parsed)
    }
    type roomRow struct {
        UserID   string
        RoomID   string
        RoomName string
    }
    studentRooms := make(map[string]roomRow)
    supervisorRooms := make(map[string]roomRow)

    if len(uuidUserIDs) > 0 {
        var studentRows []roomRow
        if err := a.DB.Table("room_students AS rs").
            Select("rs.user_id_ref AS user_id, r.id AS room_id, r.name AS room_name").
            Joins("JOIN rooms r ON r.id = rs.room_id_ref").
            Where("rs.user_id_ref IN ?", uuidUserIDs).
            Order("rs.created_at ASC").
            Scan(&studentRows).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        for _, row := range studentRows {
            if _, exists := studentRooms[row.UserID]; !exists {
                studentRooms[row.UserID] = row
            }
        }

        var supervisorRows []roomRow
        if err := a.DB.Table("room_supervisors AS rs").
            Select("rs.user_id_ref AS user_id, r.id AS room_id, r.name AS room_name").
            Joins("JOIN rooms r ON r.id = rs.room_id_ref").
            Where("rs.user_id_ref IN ?", uuidUserIDs).
            Order("rs.created_at ASC").
            Scan(&supervisorRows).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        for _, row := range supervisorRows {
            if _, exists := supervisorRooms[row.UserID]; !exists {
                supervisorRooms[row.UserID] = row
            }
        }
    }

    out := make([]gin.H, 0, len(users))
    for _, u := range users {
        entry := gin.H{
            "id":         u.ID,
            "user_id":    u.ID,
            "full_name":  u.FullName,
            "email":      u.Email,
            "role":       u.Role,
            "kelas":      u.Kelas,
            "jurusan":    u.Jurusan,
            "active":     u.Active,
            "created_at": u.CreatedAt,
            "updated_at": u.UpdatedAt,
        }
        var room interface{}
        switch strings.ToLower(u.Role) {
        case "siswa":
            if r, ok := studentRooms[u.ID]; ok {
                room = gin.H{"id": r.RoomID, "name": r.RoomName}
            }
        case "pengawas":
            if r, ok := supervisorRooms[u.ID]; ok {
                room = gin.H{"id": r.RoomID, "name": r.RoomName}
            }
        default:
            if r, ok := studentRooms[u.ID]; ok {
                room = gin.H{"id": r.RoomID, "name": r.RoomName}
            } else if r, ok := supervisorRooms[u.ID]; ok {
                room = gin.H{"id": r.RoomID, "name": r.RoomName}
            }
        }
        entry["room"] = room
        out = append(out, entry)
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
    if kelasFilter != "" {
        meta["kelas"] = kelasFilter
    }
    if jurusanFilter != "" {
        meta["jurusan"] = jurusanFilter
    }
    c.JSON(http.StatusOK, gin.H{"data": out, "meta": meta})
}

func (a *AdminController) GetUser(c *gin.Context) {
    userID := c.Param("user_id")
    var u models.User
    if err := a.DB.Where("id = ?", userID).First(&u).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "id":         u.ID,
        "user_id":    u.ID,
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
    Password *FlexibleString `json:"password"`
    Role     *string `json:"role"`
    Kelas    *FlexibleString `json:"kelas"`
    Jurusan  *string `json:"jurusan"`
    Active   *bool   `json:"active"`
}

func (a *AdminController) UpdateUser(c *gin.Context) {
    userID := c.Param("user_id")
    var u models.User
    if err := a.DB.Where("id = ?", userID).First(&u).Error; err != nil {
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
        u.Kelas = req.Kelas.String()
    }
    if req.Jurusan != nil {
        u.Jurusan = *req.Jurusan
    }
    if req.Active != nil {
        u.Active = *req.Active
    }
    if req.Password != nil {
        raw := strings.TrimSpace(req.Password.String())
        if raw != "" {
            pw, err := utils.HashPassword(raw)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
                return
            }
            u.Password = pw
        }
    }

    if err := a.DB.Save(&u).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (a *AdminController) DeleteUser(c *gin.Context) {
    userID := strings.TrimSpace(c.Param("user_id"))
    if userID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
        return
    }
    err := a.DB.Transaction(func(tx *gorm.DB) error {
        // Remove monitoring status, assignments, exit codes, refresh tokens referencing the user
        if err := tx.Where("user_id_ref = ?", userID).Delete(&models.StudentStatus{}).Error; err != nil {
            return err
        }
        if err := tx.Where("user_id_ref = ?", userID).Delete(&models.RoomSupervisor{}).Error; err != nil {
            return err
        }
        if err := tx.Where("user_id_ref = ?", userID).Delete(&models.RoomStudent{}).Error; err != nil {
            return err
        }
        if err := tx.Where("user_id_ref = ?", userID).Delete(&models.ExitCode{}).Error; err != nil {
            return err
        }
        if err := tx.Where("student_user_id_ref = ?", userID).Delete(&models.ExitCode{}).Error; err != nil {
            return err
        }
        if err := tx.Where("user_id_ref = ?", userID).Delete(&models.RefreshToken{}).Error; err != nil {
            return err
        }
        if err := tx.Where("id = ?", userID).Delete(&models.User{}).Error; err != nil {
            return err
        }
        return nil
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
