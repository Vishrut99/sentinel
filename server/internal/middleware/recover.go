package middleware

import (
	"log"
	"net/http"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic: %v", r)
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(500)
				w.Write([]byte(`{"error":"INTERNAL_SERVER_ERROR","message":"something went wrong"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
