package handler

import (
	"crypto/md5"
	"encoding/json"
	"math/big"
	"net/http"
)

const base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateShortURL creates an MD5 hash of the URL (with optional salt),
// converts it into a big integer, and Base62 encodes it (max 8 characters).
func GenerateShortURL(originalURL string, salt string) string {
	hasher := md5.New()
	hasher.Write([]byte(originalURL + salt))
	hashBytes := hasher.Sum(nil)

	num := new(big.Int).SetBytes(hashBytes)
	base := big.NewInt(62)
	zero := big.NewInt(0)
	mod := &big.Int{}
	var encoded []byte

	for num.Cmp(zero) > 0 {
		num.DivMod(num, base, mod)
		encoded = append([]byte{base62Chars[mod.Int64()]}, encoded...)
	}

	if len(encoded) == 0 {
		return string(base62Chars[0])
	}
	if len(encoded) > 8 {
		return string(encoded[:8])
	}
	return string(encoded)
}

// respondJSON writes the short_url JSON payload cleanly
func RespondJSON(w http.ResponseWriter, shortURL string) {
	response := struct {
		ShortURL string `json:"short_url"`
	}{ShortURL: shortURL}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
