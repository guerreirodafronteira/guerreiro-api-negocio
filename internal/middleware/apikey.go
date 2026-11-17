package middleware

import (
	"crypto/subtle"
	"net/http"
)

func RequireAPIKey(key string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// key vazia só chega aqui em desenvolvimento (o config.Load garante isso)
		if key != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-API-Key")), []byte(key)) != 1 {
			http.Error(w, "não autorizado", http.StatusUnauthorized)
			return // não chama next: a requisição para aqui
		}
		next.ServeHTTP(w, r)
	})
}