package middleware

import (
	"net/http"

	"golang.org/x/time/rate"
)

func Rate(next http.HandlerFunc, rps int) http.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), 1)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := limiter.Wait(r.Context()); err != nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		next(w, r)
	}
}
