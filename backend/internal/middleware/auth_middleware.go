package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	authpkg "aqi-health-monitor/backend/internal/auth"
)

// Auth xác thực Bearer token: trích token, gọi verifier,
// rồi gắn keycloak_id/claims vào request context.
func Auth(verifier *authpkg.TokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		raw, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || raw == "" {
			writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
			return
		}

		claims, err := verifier.VerifyAccessToken(r.Context(), raw)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token không hợp lệ hoặc đã hết hạn.")
			return
		}

		ctx := authpkg.ContextWithClaims(r.Context(), *claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
