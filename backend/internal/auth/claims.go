package auth

import "context"

// Claims là tập claim tối thiểu mà backend cần từ JWT của Keycloak
// (khớp ARCHITECTURE.md mục 5.1): sub → keycloak_id, email, name.
type Claims struct {
	Sub   string
	Email string
	Name  string
}

type claimsContextKey struct{}

// ContextWithClaims gắn Claims đã verify vào request context.
func ContextWithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, c)
}

// ClaimsFromContext lấy Claims từ request context (đặt bởi auth middleware).
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsContextKey{}).(Claims)
	return c, ok
}
