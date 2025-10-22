package utils

import (
    "crypto/rand"
    "math/big"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // omit easily confused chars

func GenerateCode(n int) (string, error) {
    if n <= 0 {
        n = 6
    }
    b := make([]byte, n)
    for i := 0; i < n; i++ {
        idxBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
        if err != nil {
            return "", err
        }
        b[i] = codeAlphabet[idxBig.Int64()]
    }
    return string(b), nil
}

