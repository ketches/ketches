package handler

import (
	"log"
	"net/http"
	"time"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s]\t%s\t%s\n", r.Method, r.URL.Path, r.URL.Scheme)
		now := time.Now()
		next.ServeHTTP(w, r)
		elapsed := time.Since(now)
		log.Printf("[%s]\t%s\t%s\t elapsed: %d\n", r.Method, r.URL.Path, r.URL.Scheme, elapsed)
	})
}
