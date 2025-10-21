package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(plain string) (string, error) {
    hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hashed), nil
}

func CheckPassword(hashed, plain string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

