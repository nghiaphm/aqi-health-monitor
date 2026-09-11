# Sequence Diagram — `GET /api/v1/me`

> Nguồn: implementation Bước 7 — Mục 1 (`task/CURRENT-TASK.md`) và contract `docs/09-API_CONTRACT.md` mục 4.1.
> Mục đích: trực quan hóa luồng xử lý `GET /api/v1/me` xuyên các tầng Client → Middleware → Handler → Service → Repository → Database/Keycloak, kèm tên file thực tế của từng participant.

## 1. Sơ đồ Mermaid

```mermaid
sequenceDiagram
    autonumber
    participant C as Client Next.js<br/>(dự kiến lib/api/client.ts)
    participant API as cmd/api/main.go
    participant MW as internal/middleware/<br/>auth_middleware.go
    participant TV as internal/auth/<br/>token_verifier.go
    participant JK as internal/auth/<br/>jwks_client.go
    participant KC as Keycloak<br/>(JWKS endpoint)
    participant H as internal/handler/<br/>user_handler.go
    participant S as internal/service/<br/>user_sync_service.go
    participant UR as internal/repository/<br/>user_repo.go
    participant HP as internal/repository/<br/>health_profile_repo.go
    participant AT as internal/repository/<br/>alert_threshold_repo.go
    participant LR as internal/repository/<br/>location_repo.go
    participant DB as PostgreSQL<br/>(app DB)

    C->>API: GET /api/v1/me<br/>Authorization: Bearer JWT
    API->>MW: Auth(verifier, userHandler.GetMe)
    MW->>MW: đọc header Authorization, CutPrefix "Bearer "

    alt Thiếu / sai định dạng token
        MW-->>C: 401 {error:{code:"UNAUTHORIZED"}}
    else Có Bearer token
        MW->>TV: VerifyAccessToken(ctx, rawToken)
        TV->>JK: KeySet(ctx)

        alt JWKS còn trong cache (TTL 15 phút)
            JK-->>TV: jwk.Set đã cache
        else Cache miss / hết TTL
            JK->>KC: GET {issuer}/protocol/openid-connect/certs
            KC-->>JK: JWKS (thuật toán RS256, có kid)
            JK-->>TV: jwk.Set (đã cache lại)
        end

        TV->>TV: jwt.Parse(WithKeySet, WithValidate, WithIssuer)

        alt Chữ ký sai / hết hạn exp / sai iss / thiếu sub
            TV-->>MW: error
            MW-->>C: 401 {error:{code:"UNAUTHORIZED"}}
        else Token hợp lệ
            TV-->>MW: Claims{Sub, Email, Name}
            MW->>H: next.ServeHTTP(ctx đã gắn claims)
            H->>S: GetUserState(ctx, claims)

            S->>UR: FindByKeycloakID(ctx, claims.Sub)

            alt User đã tồn tại
                UR->>DB: SELECT ... FROM users WHERE keycloak_id = $1
                DB-->>UR: row
                UR-->>S: *models.User
            else Chưa tồn tại (sql.ErrNoRows)
                S->>UR: Create(keycloak_id, email, full_name)
                UR->>DB: INSERT INTO users ... RETURNING ...
                DB-->>UR: row mới (id UUID, timestamps)
                UR-->>S: *models.User (mới)
                Note over S,UR: Race 2 request đầu tiên: Create lỗi unique keycloak_id → đọc lại FindByKeycloakID, không tạo user trùng
            end

            S->>HP: FindByUserID(user.id)
            HP->>DB: SELECT ... FROM health_profiles WHERE user_id = $1

            alt Chưa có hồ sơ sức khỏe
                DB-->>HP: 0 row
                HP-->>S: sql.ErrNoRows → profile = nil
            else Có hồ sơ sức khỏe
                DB-->>HP: row
                HP-->>S: *models.HealthProfile
            end

            S->>AT: FindByUserID(user.id)
            AT->>DB: SELECT ... FROM alert_thresholds<br/>WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 1

            alt Chưa có ngưỡng cảnh báo
                DB-->>AT: 0 row
                AT-->>S: sql.ErrNoRows → threshold = nil
            else Có ngưỡng cảnh báo
                DB-->>AT: row
                AT-->>S: *models.AlertThreshold
            end

            S->>LR: FindAllByUserID(user.id)
            LR->>DB: SELECT ..., ST_Y(geom::geometry), ST_X(geom::geometry) FROM user_locations WHERE user_id = $1
            DB-->>LR: 0..n rows
            LR-->>S: []models.UserLocation

            S-->>H: (*UserState, error)

            alt Lỗi DB / hệ thống ở bất kỳ repo nào
                H-->>C: 500 {error:{code:"INTERNAL_ERROR"}}
            else Gộp state thành công
                H->>H: map DTO: timestamp ISO 8601 UTC, full_name/city nullable
                H-->>C: 200 OK UserState (shape mục 4.1 API-CONTRACT.md)
            end
        end
    end
```

## 2. Participant ↔ file thực tế & trách nhiệm

| Participant | File thực tế | Trách nhiệm |
| --- | --- | --- |
| Client Next.js | `frontend/lib/api/client.ts` (dự kiến; chưa implement ở Bước 7) | Gọi `GET /api/v1/me` kèm `Authorization: Bearer <JWT>` lấy từ NextAuth, nhận `UserState`. |
| `cmd/api/main.go` | `backend/cmd/api/main.go` | Nạp config, mở pool PostgreSQL, dựng `JWKSClient`/`TokenVerifier`/repos/service/handler; đăng ký route `GET /api/v1/me` bọc `middleware.Auth`. |
| `auth_middleware.go` | `backend/internal/middleware/auth_middleware.go` | Trích Bearer token từ header; gọi `VerifyAccessToken`; trả `401 UNAUTHORIZED` nếu thiếu/sai token; gắn `Claims` vào request context rồi chuyển tiếp handler. |
| `token_verifier.go` | `backend/internal/auth/token_verifier.go` | Verify chữ ký JWT qua JWKS, validate `exp`/`iss`, yêu cầu có `sub`; trích `sub`/`email`/`name` thành `Claims`. |
| `jwks_client.go` | `backend/internal/auth/jwks_client.go` | Fetch và cache bộ khóa công khai từ `{issuer}/protocol/openid-connect/certs` (TTL 15 phút), tránh gọi Keycloak mỗi request. |
| Keycloak (JWKS endpoint) | Hệ thống ngoài (chưa có Realm/Client xác nhận) | Phát hành JWT và công bố khóa công khai (JWKS) để backend verify chữ ký. |
| `claims.go` | `backend/internal/auth/claims.go` | Định nghĩa `Claims{Sub, Email, Name}` + helper gắn/đọc claims trong context. |
| `user_handler.go` | `backend/internal/handler/user_handler.go` | `GetMe()` đọc claims từ context, gọi service, map sang DTO đúng shape mục 4.1, ghi JSON; trả `500 INTERNAL_ERROR` khi service lỗi. |
| `user_sync_service.go` | `backend/internal/service/user_sync_service.go` | `FindOrCreate` (upsert theo `keycloak_id`, chống tạo trùng khi race) + `GetUserState` gom aggregate; coi thiếu profile/threshold/location là trạng thái hợp lệ (`nil`/`[]`). |
| `user_repo.go` | `backend/internal/repository/user_repo.go` | `FindByKeycloakID` (trả `sql.ErrNoRows` nếu chưa có) và `Create` (`INSERT ... RETURNING` trả user mới). |
| `health_profile_repo.go` | `backend/internal/repository/health_profile_repo.go` | Bản đọc `FindByUserID` trên bảng `health_profiles` (read-only ở Mục 1). |
| `alert_threshold_repo.go` | `backend/internal/repository/alert_threshold_repo.go` | Bản đọc `FindByUserID` trên bảng `alert_thresholds`, `ORDER BY updated_at DESC LIMIT 1` (read-only ở Mục 1). |
| `location_repo.go` | `backend/internal/repository/location_repo.go` | Bản đọc `FindAllByUserID`; chuyển cột `geom` (PostGIS) sang `latitude`/`longitude` qua `ST_Y(geom::geometry)` / `ST_X(geom::geometry)`. |
| PostgreSQL (app DB) | `backend/migrations/000002–000005` | Bảng `users`, `health_profiles`, `alert_thresholds`, `user_locations` (PostGIS `GEOGRAPHY(Point,4326)`). |

## 3. Nhánh rẽ quan trọng

- **Token:** thiếu/sai định dạng → `401 UNAUTHORIZED` ngay ở middleware (không chạm DB); token sai chữ ký/hết hạn/sai `iss` → `401 UNAUTHORIZED` ở verifier.
- **FindOrCreate:** user đã tồn tại → dùng row cũ; chưa tồn tại → tạo mới. Hai request đồng thời lần đầu gây lỗi unique `keycloak_id` sẽ được xử lý bằng đọc lại thay vì tạo trùng (đúng invariant `BUSINESS-RULES.md` mục 0).
- **Dữ liệu phụ:** thiếu `health_profile` → `health_profile: null`; thiếu `alert_threshold` → `alert_threshold: null`; không có location → `locations: []`. Đây là trạng thái hợp lệ, **không** trả `PROFILE_NOT_FOUND` (theo `docs/09-API_CONTRACT.md` mục 4.1).
- **Lỗi hệ thống:** bất kỳ repo nào lỗi DB → handler trả `500 INTERNAL_ERROR` (mã chung cho 5xx, chưa định nghĩa riêng trong contract).
- **Định dạng:** mọi timestamp trả về ISO 8601 UTC (`RFC3339`, hậu tố `Z`) qua `formatUTC`; `full_name`/`city` nullable.

## 4. Ghi chú

- Bước 7 Mục 1 mới implement backend; participant `Client` là dự kiến (Mục 6 — Frontend nối dây sẽ làm).
- Realm/Client Keycloak thật chưa được xác nhận (rủi ro trong `task/TASKS.md`). Definition of Done của Mục 1 đã được test bằng scratch PostgreSQL 16 + harness OIDC tạm mô phỏng JWKS/token; khi có Keycloak thật chỉ cần trỏ `KEYCLOAK_ISSUER` đúng Realm và đảm bảo client scope trả claim `email`.
