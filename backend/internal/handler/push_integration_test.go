//go:build integration

package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

	authpkg "aqi-health-monitor/backend/internal/auth"
	cfgpkg "aqi-health-monitor/backend/internal/config"
	dbpkg "aqi-health-monitor/backend/internal/database"
	mwpkg "aqi-health-monitor/backend/internal/middleware"
	"aqi-health-monitor/backend/internal/repository"
	"aqi-health-monitor/backend/internal/service"
)

func TestPushSubscriptionsIntegration_EndToEndDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../../../.env")

	cfg := cfgpkg.Load()
	db, err := dbpkg.OpenPostgres(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		t.Fatalf("failed to connect postgres (%s:%s/%s): %v", cfg.DBHost, cfg.DBPort, cfg.DBName, err)
	}
	defer db.Close()

	// RSA keypair + local JWKS server để test đầy đủ middleware Auth + JWT verify.
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	pubJWK, err := jwk.FromRaw(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("jwk from pub: %v", err)
	}
	_ = pubJWK.Set(jwk.KeyIDKey, "test-kid")
	_ = pubJWK.Set(jwk.AlgorithmKey, jwa.RS256)
	_ = pubJWK.Set(jwk.KeyUsageKey, "sig")
	jwksPayload, err := json.Marshal(map[string]any{"keys": []any{pubJWK}})
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/protocol/openid-connect/certs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksPayload)
	}))
	defer jwksSrv.Close()

	issuer := jwksSrv.URL
	jwksClient := authpkg.NewJWKSClient(issuer)
	verifier := authpkg.NewTokenVerifier(issuer, jwksClient)

	privJWK, err := jwk.FromRaw(privKey)
	if err != nil {
		t.Fatalf("jwk from priv: %v", err)
	}
	_ = privJWK.Set(jwk.KeyIDKey, "test-kid")

	signToken := func(sub, email, name string) string {
		tok, buildErr := jwt.NewBuilder().
			Issuer(issuer).
			Subject(sub).
			IssuedAt(time.Now().UTC()).
			Expiration(time.Now().UTC().Add(10*time.Minute)).
			Claim("email", email).
			Claim("name", name).
			Build()
		if buildErr != nil {
			t.Fatalf("build jwt: %v", buildErr)
		}
		signed, signErr := jwt.Sign(tok, jwt.WithKey(jwa.RS256, privJWK))
		if signErr != nil {
			t.Fatalf("sign jwt: %v", signErr)
		}
		return string(signed)
	}

	userRepo := repository.NewUserRepo(db)
	profileRepo := repository.NewHealthProfileRepo(db)
	thresholdRepo := repository.NewAlertThresholdRepo(db)
	locationRepo := repository.NewLocationRepo(db)
	pushRepo := repository.NewPushSubscriptionRepo(db)

	syncer := service.NewUserSyncer(userRepo, profileRepo, thresholdRepo, locationRepo)
	pushSvc := service.NewPushSubscriptionService(syncer, pushRepo)
	pushHandler := NewPushHandler(pushSvc)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/push-subscriptions", mwpkg.Auth(verifier, http.HandlerFunc(pushHandler.Upsert)))
	mux.Handle("DELETE /api/v1/push-subscriptions", mwpkg.Auth(verifier, http.HandlerFunc(pushHandler.Deactivate)))

	apiSrv := httptest.NewServer(mwpkg.CORS("http://localhost:3000", mux))
	defer apiSrv.Close()

	suffix := time.Now().UTC().Format("20060102150405.000000000")
	kcUserA := "it-push-user-a-" + suffix
	kcUserB := "it-push-user-b-" + suffix
	endpoint := "https://fcm.googleapis.com/fcm/send/integration-test-" + suffix
	p256dh1 := "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM"
	auth1 := "tBHItJI5svbpez7KI4CCXg"
	p256dh2 := "BOcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM"
	auth2 := "cm90YXRlZF9hdXRoX2tleQ"

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM users WHERE keycloak_id IN ($1, $2)`, kcUserA, kcUserB)
	})

	tokA := signToken(kcUserA, "user-a-"+suffix+"@example.com", "User A")
	tokB := signToken(kcUserB, "user-b-"+suffix+"@example.com", "User B")

	doReq := func(method, token string, body any) *http.Response {
		b, _ := json.Marshal(body)
		req, reqErr := http.NewRequest(method, apiSrv.URL+"/api/v1/push-subscriptions", bytes.NewReader(b))
		if reqErr != nil {
			t.Fatalf("new request: %v", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			t.Fatalf("do request: %v", doErr)
		}
		return resp
	}

	// 0. CORS preflight from frontend origin http://localhost:3000
	preflightReq, err := http.NewRequest(http.MethodOptions, apiSrv.URL+"/api/v1/push-subscriptions", nil)
	if err != nil {
		t.Fatalf("new preflight request: %v", err)
	}
	preflightReq.Header.Set("Origin", "http://localhost:3000")
	preflightReq.Header.Set("Access-Control-Request-Method", "POST")
	preflightReq.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	preflightResp, err := http.DefaultClient.Do(preflightReq)
	if err != nil {
		t.Fatalf("preflight request: %v", err)
	}
	preflightResp.Body.Close()
	if preflightResp.StatusCode != http.StatusNoContent || preflightResp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("unexpected preflight response: status=%d allow-origin=%q", preflightResp.StatusCode, preflightResp.Header.Get("Access-Control-Allow-Origin"))
	}

	// 1. Unauthorized without token
	respUnauth := doReq(http.MethodPost, "", map[string]string{
		"endpoint": endpoint, "p256dh_key": p256dh1, "auth_key": auth1,
	})
	respUnauth.Body.Close()
	if respUnauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", respUnauth.StatusCode)
	}

	// 2. First registration by User A
	resp1 := doReq(http.MethodPost, tokA, map[string]string{
		"endpoint": endpoint, "p256dh_key": p256dh1, "auth_key": auth1,
	})
	var dto1 map[string]any
	_ = json.NewDecoder(resp1.Body).Decode(&dto1)
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on first register, got %d: %v", resp1.StatusCode, dto1)
	}
	subID, _ := dto1["id"].(string)
	createdAt, _ := dto1["created_at"].(string)
	if subID == "" || dto1["is_active"] != true || dto1["last_failed_at"] != nil {
		t.Fatalf("unexpected first register response: %v", dto1)
	}
	if _, hasKey := dto1["p256dh_key"]; hasKey {
		t.Fatalf("p256dh_key must not be echoed")
	}

	// Verify row directly in DB
	var dbUserID, dbP256dh, dbAuth string
	var dbActive bool
	var dbFailedAt sql.NullTime
	err = db.QueryRow(`SELECT user_id, p256dh_key, auth_key, is_active, last_failed_at FROM push_subscriptions WHERE id = $1`, subID).
		Scan(&dbUserID, &dbP256dh, &dbAuth, &dbActive, &dbFailedAt)
	if err != nil {
		t.Fatalf("query push_subscriptions: %v", err)
	}
	if !dbActive || dbFailedAt.Valid || dbP256dh != p256dh1 || dbAuth != auth1 {
		t.Fatalf("unexpected DB state after register: active=%v failed=%v p256dh=%s auth=%s", dbActive, dbFailedAt.Valid, dbP256dh, dbAuth)
	}

	// 3. Simulate prior 410 failure in DB then re-register same user with rotated keys
	_, err = db.Exec(`UPDATE push_subscriptions SET is_active = false, last_failed_at = now() - interval '1 hour' WHERE id = $1`, subID)
	if err != nil {
		t.Fatalf("simulate failure update: %v", err)
	}

	resp2 := doReq(http.MethodPost, tokA, map[string]string{
		"endpoint": endpoint, "p256dh_key": p256dh2, "auth_key": auth2,
	})
	var dto2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&dto2)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on re-register, got %d: %v", resp2.StatusCode, dto2)
	}
	if dto2["id"] != subID || dto2["created_at"] != createdAt || dto2["is_active"] != true || dto2["last_failed_at"] != nil {
		t.Fatalf("re-register did not preserve id/created_at or reset status: %v", dto2)
	}

	err = db.QueryRow(`SELECT p256dh_key, auth_key, is_active, last_failed_at FROM push_subscriptions WHERE id = $1`, subID).
		Scan(&dbP256dh, &dbAuth, &dbActive, &dbFailedAt)
	if err != nil {
		t.Fatalf("query after re-register: %v", err)
	}
	if !dbActive || dbFailedAt.Valid || dbP256dh != p256dh2 || dbAuth != auth2 {
		t.Fatalf("unexpected DB state after re-register: active=%v failed=%v p256dh=%s auth=%s", dbActive, dbFailedAt.Valid, dbP256dh, dbAuth)
	}

	// 4. Cross-user endpoint conflict -> reassigns ownership to User B, count stays 1
	resp3 := doReq(http.MethodPost, tokB, map[string]string{
		"endpoint": endpoint, "p256dh_key": p256dh1, "auth_key": auth1,
	})
	var dto3 map[string]any
	_ = json.NewDecoder(resp3.Body).Decode(&dto3)
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on cross-user reassign, got %d: %v", resp3.StatusCode, dto3)
	}

	var count int
	var newOwnerKC string
	err = db.QueryRow(`SELECT count(*) FROM push_subscriptions WHERE endpoint = $1`, endpoint).Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 row for endpoint, got count=%d err=%v", count, err)
	}
	err = db.QueryRow(`SELECT u.keycloak_id FROM push_subscriptions ps JOIN users u ON u.id = ps.user_id WHERE ps.endpoint = $1`, endpoint).
		Scan(&newOwnerKC)
	if err != nil || newOwnerKC != kcUserB {
		t.Fatalf("expected reassigned owner %s, got %s (err=%v)", kcUserB, newOwnerKC, err)
	}

	// 5. User A cannot soft-disable User B's subscription
	respDelA := doReq(http.MethodDelete, tokA, map[string]string{"endpoint": endpoint})
	respDelA.Body.Close()
	if respDelA.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when User A deletes User B's endpoint, got %d", respDelA.StatusCode)
	}

	// 6. User B soft-disables their subscription -> is_active = false in DB (not deleted)
	respDelB := doReq(http.MethodDelete, tokB, map[string]string{"endpoint": endpoint})
	var dtoDelB map[string]any
	_ = json.NewDecoder(respDelB.Body).Decode(&dtoDelB)
	respDelB.Body.Close()
	if respDelB.StatusCode != http.StatusOK || dtoDelB["is_active"] != false {
		t.Fatalf("expected 200 with is_active=false, got %d: %v", respDelB.StatusCode, dtoDelB)
	}

	err = db.QueryRow(`SELECT is_active FROM push_subscriptions WHERE endpoint = $1`, endpoint).Scan(&dbActive)
	if err != nil || dbActive {
		t.Fatalf("expected DB is_active=false after DELETE, got active=%v err=%v", dbActive, err)
	}
}
