package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

const jwksCacheTTL = 15 * time.Minute

// JWKSClient fetch + cache bộ khóa công khai từ Keycloak
// (endpoint: {issuer}/protocol/openid-connect/certs).
type JWKSClient struct {
	mu        sync.Mutex
	client    *http.Client
	certsURL  string
	cachedSet jwk.Set
	cachedAt  time.Time
}

func NewJWKSClient(issuer string) *JWKSClient {
	return &JWKSClient{
		client:   &http.Client{Timeout: 10 * time.Second},
		certsURL: issuer + "/protocol/openid-connect/certs",
	}
}

// KeySet trả về jwk.Set, refresh tối đa 1 lần / jwksCacheTTL.
func (c *JWKSClient) KeySet(ctx context.Context) (jwk.Set, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedSet != nil && time.Since(c.cachedAt) < jwksCacheTTL {
		return c.cachedSet, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.certsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jwks body: %w", err)
	}
	set, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}

	c.cachedSet = set
	c.cachedAt = time.Now()
	return set, nil
}
