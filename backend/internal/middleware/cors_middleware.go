package middleware

import (
	"net/http"
	"strings"
)

// CORS xử lý preflight OPTIONS và gắn Access-Control-Allow-* cho origin được phép
// (mặc định http://localhost:3000 cho Next.js frontend).
func CORS(allowedOrigins string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{})
	allowAll := false
	for _, raw := range strings.Split(allowedOrigins, ",") {
		o := strings.TrimSpace(raw)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
		}
		allowed[o] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")

		if origin != "" {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
