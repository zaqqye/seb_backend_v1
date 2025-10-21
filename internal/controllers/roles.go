package controllers

var allowedRoles = map[string]struct{}{
    "admin":    {},
    "pengawas": {},
    "siswa":    {},
}

func IsValidRole(role string) bool {
    _, ok := allowedRoles[role]
    return ok
}

