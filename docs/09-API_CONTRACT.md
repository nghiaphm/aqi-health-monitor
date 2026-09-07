# API Contract — AQI Health Monitor

> **Trạng thái**: Bước 6 — API & Integration Contract Design (✅ Xong). Contract được chốt cho **4 endpoint của Vertical Slice đầu tiên**, không phải toàn bộ API sản phẩm.
>
> - Tên file vật lý: `docs/09-API_CONTRACT.md` — trong bảng `TASKS.md`/`CURRENT-TASK.md` gọi tắt là `API-CONTRACT.md`.
> - Nguồn đối chiếu bắt buộc (không tự suy đoán field/quy tắc nghiệp vụ ngoài các file này):
>   - `docs/02-BUSINESS-RULES.md` — con số, invariant, ngưỡng cảnh báo, cooldown/dedup, khoảng cách 20km, stale 2h/3h
>   - `docs/03-USER-FLOWS.md` — 7 luồng user & các nhánh rẽ phải xử lý được trên Frontend
>   - `docs/06-SYSTEM-FLOW.md` — thứ tự vận hành & các giai đoạn (onboarding, dashboard, worker nền)
>   - `docs/07-DATABASE_DESIGN.md` — khớp 1:1 migration `000001`→`000011`
>   - `docs/08-SA.docx` — kiến trúc hệ thống (modular monolith, `/api/v1`, Redis cache-aside, tầng handler→service→repository)
> - Khi 1 quyết định trong file này mâu thuẫn với tài liệu gốc: **sửa tài liệu gốc trước**, sau đó mới sửa contract — đúng nguyên tắc "tài liệu dẫn dắt code".

## Mục lục

1. [Tổng quan endpoints](#1-tổng-quan-endpoints)
2. [Quy ước chung](#2-quy-ước-chung)
   - [2.1 Versioning & Base URL](#21-versioning--base-url)
   - [2.2 Authentication](#22-authentication)
   - [2.3 Content-Type & quy tắc chung về JSON](#23-content-type--quy-tắc-chung-về-json)
   - [2.4 ID](#24-id)
   - [2.5 Timestamp](#25-timestamp)
   - [2.6 Tọa độ & khoảng cách](#26-tọa-độ--khoảng-cách)
   - [2.7 Enum value](#27-enum-value)
   - [2.8 AQI level code](#28-aqi-level-code)
   - [2.9 Format response thành công](#29-format-response-thành-công)
   - [2.10 Format response lỗi](#210-format-response-lỗi)
   - [2.11 Bảng mã lỗi nghiệp vụ & HTTP status](#211-bảng-mã-lỗi-nghiệp-vụ--http-status)
   - [2.12 Ký hiệu "chuỗi file tương tác"](#212-ký-hiệu-chuỗi-file-tương-tác)
3. [DTO tham chiếu dùng chung](#3-dto-tham-chiếu-dùng-chung)
4. [Endpoint chi tiết](#4-endpoint-chi-tiết)
   - [4.1 GET /api/v1/me](#41-get-apiv1me)
   - [4.2 POST /api/v1/health-profile](#42-post-apiv1health-profile)
   - [4.3 POST /api/v1/locations](#43-post-apiv1locations)
   - [4.4 GET /api/v1/aqi/current](#44-get-apiv1aqicurrent)
5. [Đối chiếu với USER-FLOWS (7 flow)](#5-đối-chiếu-với-user-flows-7-flow)
6. [Quyết định & backlog sau Bước 6](#6-quyết-định--backlog-sau-bước-6)

---

## 1. Tổng quan endpoints

| # | Method & Path | Mục đích | User Flow gốc |
| --- | --- | --- | --- |
| 1 | `GET /api/v1/me` | Trạng thái hiện tại của user: identity + health profile + threshold + danh sách vị trí (đủ để FE quyết định Onboarding/Dashboard) | Flow 1, 2, 5 |
| 2 | `POST /api/v1/health-profile` | Upsert hồ sơ sức khỏe + ngưỡng cảnh báo theo user (dùng cho Onboarding lẫn chỉnh hồ sơ/ngưỡng sau này) | Flow 2, 3 |
| 3 | `POST /api/v1/locations` | Upsert vị trí theo `label` (`current`/`home`) — GPS hoặc nhập tay | Flow 2 |
| 4 | `GET /api/v1/aqi/current` | AQI hiện tại tại 1 vị trí đã lưu của user (màn hình Dashboard) | Flow 5 |

Ghi chú phạm vi: Push subscription (Flow 4), History (Flow 6), Alerts list (Flow 7) **không** thuộc 4 endpoint này — xem mục [6](#6-quyết-định--backlog-sau-bước-6).

---

## 2. Quy ước chung

### 2.1 Versioning & Base URL

- Mọi request đi qua prefix **`/api/v1`** (versioning theo path — theo `08-SA.docx` mục 3.2).
- Base URL ở local: `http://localhost:8000` (backend-api). Frontend gọi qua `NEXT_PUBLIC_API_URL`.
- Không dùng header `Accept-Version` ở MVP — version path là chuẩn duy nhất.
- Trừ khi có ghi chú riêng, mọi response là `application/json; charset=utf-8`.

### 2.2 Authentication

- Tất cả 4 endpoint yêu cầu **Bearer token** (JWT do Keycloak cấp qua NextAuth ở Frontend):

```
Authorization: Bearer <access_token>
```

- Backend verify JWT qua JWKS (không tự lưu password). Claims dùng: `sub` → `keycloak_id` nội bộ, `email`, `name` (`08-SA.docx` mục 5.1).
- Luồng đồng bộ user: nếu `keycloak_id` chưa có trong `users`, backend **upsert tự động** rồi gắn `user_id` nội bộ vào request context (`06-SYSTEM-FLOW.md` Giai đoạn 1) — các handler/service chỉ làm việc với `user_id` nội bộ.
- Token thiếu/sai/đã hết hạn → `401 UNAUTHORIZED`.

### 2.3 Content-Type & quy tắc chung về JSON

- Yêu cầu: gửi kèm header `Content-Type: application/json` với body là JSON object.
- Key dùng **`snake_case`**.
- Kiểu dữ liệu theo JSON chuẩn: string, number, boolean, object, array, `null`. **Không** dùng `null` thay cho trường bắt buộc bị thiếu — thiếu trường bắt buộc = `400 VALIDATION_ERROR`.
- Trường optional khi không có giá trị: trả `null` (giữ key) — trừ trường hợp đã ghi rõ "bỏ key".

### 2.4 ID

- Mọi resource ID (`user`, `health_profile`, `alert_threshold`, `location`, `station`) là **UUID v4**, truyền/nhận dưới dạng **string**.
- Format chuẩn: `xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx` (hex, lowercase). Backend sinh bằng `gen_random_uuid()` (DB) / `google/uuid` (Go).
- Không expose ID gốc bên thứ ba (VD: `waqi_station_id`) làm ID của API — theo đúng `07-DATABASE_DESIGN.md` mục 3.5.

### 2.5 Timestamp

- Mọi timestamp theo **ISO 8601 / RFC 3339**, **UTC**, luôn kèm ký tự `Z` (không dùng offset giờ địa phương).
- Ví dụ: `2026-09-07T02:00:00Z`.
- Quy ước tên field: tạo/cập nhật → `created_at`, `updated_at`; thời điểm đo AQI → `measured_at`.
- Backend có thể trả phần phân số giây (millisecond) — Frontend nên parse bằng `Date.parse` (chuẩn) thay vì string thủ công.
- So sánh "độ mới của dữ liệu" ở Frontend luôn tính trên UTC.

### 2.6 Tọa độ & khoảng cách

- `latitude`, `longitude`: số thập phân theo độ (WGS84).
- Phạm vi hợp lệ: `latitude ∈ [-90, 90]`, `longitude ∈ [-180, 180]`. Ngoài phạm vi → `400 VALIDATION_ERROR` (theo `08-SA.docx` mục 5.2 — bắt buộc validate tọa độ trước khi chạm PostGIS).
- Lưu trữ làm tròn **6 chữ số thập phân**.
- Lưu ý mapping DB: PostGIS `GEOGRAPHY(Point, 4326)` lưu theo thứ tự `(lng, lat)` — nhưng contract API **luôn** trao đổi dưới dạng object `{ "latitude": …, "longitude": … }`, việc đổi thứ tự là chi tiết implementation của repository.
- `distance_m`: khoảng cách, đơn vị **mét**, kiểu integer.

### 2.7 Enum value

| Enum | Giá trị hợp lệ | Ghi chú |
| --- | --- | --- |
| `condition_type` | `none` \| `asthma` \| `allergic_rhinitis` | `02-BUSINESS-RULES.md` mục 1 |
| `age_group` | `child` \| `adult` \| `elderly` | |
| `sensitivity_level` | `normal` \| `high` | Mặc định `normal` nếu không gửi |
| `location.label` | `current` \| `home` | Upsert theo `(user_id, label)` |
| `location.city` | `null` \| free string ≤ 100 ký tự | Khuyến nghị `hanoi` \| `ho-chi-minh-city` (khớp scope MVP); **không** ràng buộc cứng — DB không CHECK |
| `aqi.level` | xem [2.8](#28-aqi-level-code) | Backend tính từ giá trị AQI |

### 2.8 AQI level code

Mapping AQI → level machine code (định nghĩa ở đây để Backend và Frontend dùng chung 1 bộ giá trị; màu sắc/label VN Frontend map theo token `docs/05-DESIGN-DECISIONS.md`):

| Khoảng AQI | `level` | Nhãn VN | Token màu FE |
| --- | --- | --- | --- |
| 0–50 | `good` | Tốt | `aqi-good` |
| 51–100 | `moderate` | Trung bình | `aqi-moderate` |
| 101–150 | `unhealthy_sensitive` | Kém | `aqi-unhealthySensitive` |
| 151–200 | `unhealthy` | Xấu | `aqi-unhealthy` |
| 201–300 | `very_unhealthy` | Rất xấu | `aqi-veryUnhealthy` |
| > 300 | `hazardous` | Nguy hiểm | `aqi-hazardous` |

> Lưu ý: code API dùng `snake_case` (`unhealthy_sensitive`, `very_unhealthy`) — khác tên token camelCase ở FE. Ranh giới 50/100/150/200/300 phải khớp `RESEARCH.md`, `DESIGN-DECISIONS.md` và bảng này (đổi 1 nơi → đổi cả 3 nơi).

### 2.9 Format response thành công

- Response 2xx trả **trực tiếp resource JSON** — **không** bọc thêm envelope `{data: …}` hay `{success: …}`.
- 4 endpoint của slice đều là thao tác trả 1 resource:
  - `GET /me`, `GET /aqi/current` → `200 OK` + object.
  - `POST /health-profile`, `POST /locations` → **`200 OK`** + object kết quả (không dùng `201`): vì đây là **upsert idempotent** (create lẫn update), client không cần phân biệt "mới tạo" hay "đã cập nhật"; body phản ánh trạng thái cuối sau khi lưu.
- Không có response rỗng (`204`) trong phạm vi 4 endpoint này.

### 2.10 Format response lỗi

- Mọi lỗi (không phải 2xx) trả về **1 envelope duy nhất**:

```json
{
  "error": {
    "code": "THRESHOLD_TOO_LOW",
    "message": "Ngưỡng cảnh báo không được dưới 50 (AQI dưới mức 'Trung bình' sẽ gây cảnh báo liên tục).",
    "details": [
      { "field": "custom_threshold_aqi", "reason": "min=50" }
    ]
  }
}
```

| Field | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| `error.code` | string | ✅ | Mã lỗi máy đọc được (bảng [2.11](#211-bảng-mã-lỗi-nghiệp-vụ--http-status)). FE branch theo `code`, không theo message. |
| `error.message` | string | ✅ | Mô tả ngắn cho user (tiếng Việt). |
| `error.details` | array | ⬜ | Chỉ hiện diện khi `VALIDATION_ERROR` (1 phần tử cho mỗi field sai); gồm `field` + `reason`. |

Quy tắc cho Frontend: client API layer **giải mã envelope trước**, map `code` → UI state (VD: `NO_DATA` → màn "chưa có dữ liệu", không phải toast lỗi chung) — xem chi tiết từng endpoint.

### 2.11 Bảng mã lỗi nghiệp vụ & HTTP status

Mọi mã lỗi dưới đây đều **đối chiếu được với tài liệu gốc** — không đặt thêm case ngoài tài liệu.

| Code | HTTP | Ý nghĩa | Endpoint nào trả (trong 4 endpoint) | Nguồn tài liệu |
| --- | --- | --- | --- | --- |
| `UNAUTHORIZED` | 401 | Token thiếu / không hợp lệ / hết hạn — không xác định được user | Tất cả | `06-SYSTEM-FLOW.md` GĐ1; `03-USER-FLOWS.md` Flow 1 ("đăng nhập thất bại") |
| `VALIDATION_ERROR` | 400 | Payload sai cấu trúc/kiểu/giá trị: sai enum, thiếu field bắt buộc, `lat`/`lng` ngoài khoảng, `threshold_aqi > 500`, không phải số nguyên… Kèm `details[]`. | `POST /health-profile`, `POST /locations`, `GET /aqi/current` (query sai format) | `08-SA.docx` mục 5.2 (input validation, đặc biệt tọa độ GPS); `03-USER-FLOWS.md` Flow 3 ("giá trị nhập không hợp lệ"); CHECK `0–500` trong `07-DATABASE_DESIGN.md` mục 3.3 |
| `THRESHOLD_TOO_LOW` | 422 | `custom_threshold_aqi < 50` — yêu cầu hợp lệ về cấu trúc nhưng vi phạm ràng buộc nghiệp vụ. **Không** tự clamp lại giá trị. | `POST /health-profile` | `02-BUSINESS-RULES.md` mục 1 ("từ chối giá trị dưới 50… Trả lỗi rõ ràng cho Frontend hiển thị, không âm thầm clamp") |
| `PROFILE_NOT_FOUND` | 404 | Thao tác **yêu cầu hồ sơ đã tồn tại** nhưng user chưa có `health_profile` | Không endpoint nào trong 4 endpoint này (xem mục [6](#6-quyết-định--backlog-sau-bước-6)) — dự kiến cho endpoint profile/threshold riêng ở Bước 7/8 | `02-BUSINESS-RULES.md` mục 0 (invariant profile); `03-USER-FLOWS.md` Flow 2 |
| `LOCATION_NOT_FOUND` | 404 | `location_id` không tồn tại / không thuộc user; hoặc user **không có vị trí nào** để resolve mặc định | `GET /aqi/current` | `03-USER-FLOWS.md` Flow 5 ("chưa có vị trí đã lưu") |
| `NO_STATION_COVERAGE` | 404 | Không có trạm `is_active` nào trong bán kính **20 km** quanh vị trí → coi như "chưa có dữ liệu khu vực này" | `GET /aqi/current` | `02-BUSINESS-RULES.md` mục 3 (20 km) |
| `NO_DATA` | 404 | Có trạm hợp lệ nhưng **chưa từng có** `aqi_readings` nào (worker chưa chạy / trạm mới thêm) | `GET /aqi/current` | `03-USER-FLOWS.md` Flow 5 ("cache Redis miss và TimescaleDB chưa có dữ liệu") |
| `DATA_STALE` | 422 | Có reading nhưng **quá cũ** (muộn hơn **3 giờ** so với hiện tại) — không đủ tin cậy để trả giá trị "hiện tại" | `GET /aqi/current` | `02-BUSINESS-RULES.md` mục 3 (stale > 3h — gấp 3 chu kỳ fetch) |

Nhóm HTTP status được dùng có chủ đích:

- `400` — input sai (schema/format/enum/giá trị vùng).
- `401` — chưa xác thực.
- `404` — "thứ được hỏi đến không tồn tại/không có" (profile, location, coverage, dữ liệu chưa từng có).
- `422` — request hợp lệ nhưng **trạng thái nghiệp vụ** không cho phép xử lý (ngưỡng quá thấp, dữ liệu cũ).
- `5xx` — lỗi server (không định nghĩa code riêng ở slice này; FE hiển thị lỗi chung + retry).

### 2.12 Ký hiệu "chuỗi file tương tác"

Mỗi endpoint có 1 mục **"Chuỗi file tương tác"** liệt kê đúng thứ tự các file tham gia từ Frontend xuống DB, theo mô tả dạng:

```
[FE] file gọi API → [cmd/api] route → [middleware] auth → [handler] → [service] → [repository] → [Postgres / Redis]
```

Các file backend đã scaffold (nội dung sẽ viết ở Bước 7). Quy ước tầng theo `02-BUSINESS-RULES.md` mục 6 và `08-SA.docx` mục 3.1:

- **Handler** (`internal/handler/*.go`): parse request, validate đầu vào, đóng gói response/error envelope. **Không** chứa logic nghiệp vụ.
- **Service** (`internal/service/*.go`): toàn bộ business rules/invariant — enforce tại tầng này.
- **Repository** (`internal/repository/*_repo.go`): SQL thuần (PostGIS/TimescaleDB), trả domain model.
- **Redis** (`internal/database/redis.go`): cache-aside AQI.

---

## 3. DTO tham chiếu dùng chung

### 3.1. `Location` (user_locations)

| Field | Kiểu | Mô tả |
| --- | --- | --- |
| `id` | string (UUID) | ID vị trí |
| `label` | enum | `current` \| `home` |
| `latitude` | number | WGS84, 6 số lẻ |
| `longitude` | number | WGS84, 6 số lẻ |
| `city` | string \| null | `hanoi` \| `ho-chi-minh-city` (khuyến nghị) hoặc null |
| `created_at` | string (ISO) | Thời điểm tạo (upsert giữ nguyên) |

### 3.2. `HealthProfile` (health_profiles)

| Field | Kiểu | Mô tả |
| --- | --- | --- |
| `id` | string (UUID) | ID hồ sơ |
| `condition_type` | enum | `none` \| `asthma` \| `allergic_rhinitis` |
| `age_group` | enum | `child` \| `adult` \| `elderly` |
| `sensitivity_level` | enum | `normal` \| `high` |
| `created_at` / `updated_at` | string (ISO) | |

### 3.3. `AlertThreshold` (alert_thresholds)

| Field | Kiểu | Mô tả |
| --- | --- | --- |
| `id` | string (UUID) | ID ngưỡng |
| `threshold_aqi` | integer | Giá trị ngưỡng (0–500) |
| `is_custom` | boolean | `true` = user tự đặt; `false` = tính theo ma trận mặc định |
| `updated_at` | string (ISO) | |

### 3.4. `UserState` (aggregate của GET /me)

= user + `health_profile` + `alert_threshold` + `locations[]`. Chi tiết ở [4.1](#41-get-apiv1me).

### 3.5. `CurrentAqi` (aggregate của GET /aqi/current)

= `location` + `station` + `reading` + `advice`. Chi tiết ở [4.4](#44-get-apiv1aqicurrent).

---

## 4. Endpoint chi tiết

### 4.1 GET /api/v1/me

**Mục đích**: trả trạng thái tổng hợp của user để FE quyết định điều hướng (Onboarding vs Dashboard) và prefill các màn Profile/Settings/Locations mà không cần thêm GET endpoint trong slice này.

**Auth**: ✅ Bearer token.

**Chuỗi file tương tác**:

```
[frontend] useCurrentUser hook / lib/api/user.ts
  → GET /api/v1/me (Bearer token)
[backend]
  1. cmd/api/main.go                    — route group /api/v1, gắn auth middleware
  2. internal/middleware/auth_middleware.go
        — đọc Authorization header, verify JWT qua token_verifier.go + jwks_client.go
        — lấy claims {sub, email, name}; tìm/tạo user nội bộ; gắn user_id vào context
  3. internal/handler/user_handler.go   — GetMe(): đọc user_id từ context, gọi service, đóng envelope
  4. internal/service/user_sync_service.go — GetUserState(userID):
        a. đảm bảo user tồn tại (FindOrCreate theo keycloak_id — upsert)
        b. gom state: profile + threshold hiệu lực + danh sách locations
  5. internal/repository/
        user_repo.go                 — FindByKeycloakID / Create
        health_profile_repo.go       — FindByUserID
        alert_threshold_repo.go      — FindByUserID (lấy 1 ngưỡng duy nhất đang hiệu lực)
        location_repo.go             — FindAllByUserID
  6. PostgreSQL                       — bảng users, health_profiles, alert_thresholds, user_locations
[backend] → 200 JSON UserState (hoặc 401)
```

**Response — 200 OK**:

```json
{
  "id": "8e3a4b1f-6d20-4e6a-9b0d-2f1a7c9d3e55",
  "email": "nguyen.van.a@example.com",
  "full_name": "Nguyễn Văn A",
  "created_at": "2026-09-07T02:00:00Z",
  "health_profile": {
    "id": "b2f0a8c1-3d44-4e9a-8c10-5b7f2d9e1a20",
    "condition_type": "asthma",
    "age_group": "adult",
    "sensitivity_level": "normal",
    "created_at": "2026-09-07T02:10:00Z",
    "updated_at": "2026-09-07T02:10:00Z"
  },
  "alert_threshold": {
    "id": "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
    "threshold_aqi": 100,
    "is_custom": false,
    "updated_at": "2026-09-07T02:10:00Z"
  },
  "locations": [
    {
      "id": "6f9d0e2c-4b5a-4c7d-9e1a-2b3c4d5e6f70",
      "label": "current",
      "latitude": 21.030578,
      "longitude": 105.854208,
      "city": "hanoi",
      "created_at": "2026-09-07T02:15:00Z"
    }
  ]
}
```

**Bảng field**:

| Field | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| `id` | string (UUID) | ✅ | User nội bộ |
| `email` | string | ✅ | Đồng bộ từ Keycloak claim `email` |
| `full_name` | string \| null | ✅ (có thể null) | Từ claim `name`; `null` nếu Keycloak không có |
| `created_at` | string (ISO) | ✅ | |
| `health_profile` | `HealthProfile` \| null | ✅ | `null` = chưa Onboarding |
| `alert_threshold` | `AlertThreshold` \| null | ✅ | `null` khi chưa có profile/ngưỡng |
| `locations` | array[`Location`] | ✅ | Luôn là array; rỗng nếu chưa có vị trí |

**Nhánh FE xử lý được từ response**:

| Điều kiện | FE hành động | Flow gốc |
| --- | --- | --- |
| `health_profile == null` | Chuyển sang **Onboarding** (gồm cả user mới lần đầu đăng nhập) | Flow 1, 2 |
| `health_profile != null` và `locations.length == 0` | Vào **Dashboard hạn chế** + CTA "bổ sung vị trí" (không gọi `/aqi/current`) | Flow 2, 5 |
| `health_profile != null` và `locations.length > 0` | Vào **Dashboard đầy đủ**, prefill Profile/Settings/Locations từ aggregate | Flow 5 |

**Lỗi**: `401 UNAUTHORIZED`. (Không bao giờ trả `PROFILE_NOT_FOUND` — thiếu hồ sơ là trạng thái hợp lệ, biểu diễn bằng `health_profile: null`.)

---

### 4.2 POST /api/v1/health-profile

**Mục đích**: tạo/cập nhật hồ sơ sức khỏe và ngưỡng cảnh báo theo user.

- Dùng cho **Onboarding** (Flow 2): gửi `condition_type` + `age_group` (không gửi `custom_threshold_aqi`) → backend tính ngưỡng mặc định theo ma trận, `is_custom = false`.
- Dùng cho **chỉnh hồ sơ / chỉnh ngưỡng** (Flow 3, từ màn Settings): gửi lại **toàn bộ payload** kèm `custom_threshold_aqi` → `is_custom = true`.
- Là **upsert idempotent**: 1 user chỉ có 1 profile (DB UNIQUE) và **1 ngưỡng hiệu lực** — service phải update-in-place theo user, không tạo thêm row (`02-BUSINESS-RULES.md` mục 0). An toàn để retry.

**Auth**: ✅ Bearer token.

**Chuỗi file tương tác**:

```
[frontend] lib/api/health-profile.ts (health-profile-form.tsx, threshold-slider.tsx)
  → POST /api/v1/health-profile (Bearer token)
[backend]
  1. cmd/api/main.go                    — route /api/v1/health-profile, auth middleware
  2. internal/middleware/auth_middleware.go — verify JWT, gắn user_id
  3. internal/handler/health_profile_handler.go — Upsert():
        - decode & validate cấu trúc (enum, kiểu số, threshold trong 50–500 → VALIDATION_ERROR)
        - validate nghiệp vụ threshold < 50 → THRESHOLD_TOO_LOW (KHÔNG clamp)
  4. internal/service/health_profile_service.go — UpsertProfile(userID, payload):
        a. nếu custom_threshold_aqi == null → tính mặc định theo ma trận
           (condition_type + age_group + sensitivity_level; clamp 50–300)
        b. upsert health_profiles theo user_id (transaction)
        c. upsert alert_thresholds theo user_id — update-in-place, set is_custom tương ứng
        (ma trận 18 tổ hợp là constant trong service, không lưu DB)
  5. internal/repository/
        health_profile_repo.go      — UpsertByUserID
        alert_threshold_repo.go     — UpsertByUserID
  6. PostgreSQL                     — bảng health_profiles, alert_thresholds (1 transaction)
[backend] → 200 JSON profile + threshold (hoặc error envelope)
```

**Request body**:

```json
{
  "condition_type": "asthma",
  "age_group": "adult",
  "sensitivity_level": "high",
  "custom_threshold_aqi": 80
}
```

**Bảng field request**:

| Field | Kiểu | Bắt buộc | Ràng buộc / Mô tả |
| --- | --- | --- | --- |
| `condition_type` | enum | ✅ | `none` \| `asthma` \| `allergic_rhinitis` |
| `age_group` | enum | ✅ | `child` \| `adult` \| `elderly` |
| `sensitivity_level` | enum | ⬜ | Mặc định `normal` khi bỏ trống (MVP chưa thu thập trên UI Onboarding) |
| `custom_threshold_aqi` | integer | ⬜ | Phạm vi **50–500**. Bỏ trống → tính mặc định theo ma trận, `is_custom=false`. Gửi kèm → `is_custom=true`. |

**Response — 200 OK** (upsert, trả trạng thái cuối):

```json
{
  "id": "b2f0a8c1-3d44-4e9a-8c10-5b7f2d9e1a20",
  "condition_type": "asthma",
  "age_group": "adult",
  "sensitivity_level": "high",
  "created_at": "2026-09-07T02:10:00Z",
  "updated_at": "2026-09-07T02:10:00Z",
  "alert_threshold": {
    "id": "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
    "threshold_aqi": 80,
    "is_custom": true,
    "updated_at": "2026-09-07T02:10:00Z"
  }
}
```

Ví dụ khi **không gửi** `custom_threshold_aqi` (`asthma` + `adult` + `normal`) → backend tự tính = `100`, trả `"threshold_aqi": 100, "is_custom": false`.

**Lỗi**:

| HTTP | Code | Ví dụ điều kiện |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Sai enum, thiếu `condition_type`, `custom_threshold_aqi` = 501 hoặc không phải số nguyên |
| 422 | `THRESHOLD_TOO_LOW` | `custom_threshold_aqi` = 49 (dưới 50 — sẽ cảnh báo liên tục, không clamp) |
| 401 | `UNAUTHORIZED` | Token không hợp lệ |

**Nhánh FE xử lý**: dựa vào `error.code` hiển thị lỗi dưới field tương ứng (dùng `details[].field`); thành công → đọc `alert_threshold.is_custom`/`threshold_aqi` để hiển thị lại Settings.

---

### 4.3 POST /api/v1/locations

**Mục đích**: upsert vị trí theo `label`. Dùng cho cả GPS (label `current`) lẫn nhập địa chỉ tay (label `home`) trong Onboarding / quản lý vị trí.

**Auth**: ✅ Bearer token.

**Chuỗi file tương tác**:

```
[frontend] lib/api/locations.ts (location-picker.tsx, use-current-location.ts)
  → POST /api/v1/locations (Bearer token)
[backend]
  1. cmd/api/main.go                    — route /api/v1/locations, auth middleware
  2. internal/middleware/auth_middleware.go — verify JWT, gắn user_id
  3. internal/handler/location_handler.go — Upsert():
        - validate label ∈ {current, home}
        - validate lat ∈ [-90,90], lng ∈ [-180,180] (bắt buộc trước khi chạm PostGIS)
        - validate city: null hoặc string ≤ 100 ký tự (trim, lowercase)
  4. internal/service/location_service.go — UpsertByUserAndLabel(userID, payload):
        - enforce invariant "1 user tối đa 1 vị trí mỗi label" → upsert theo (user_id, label),
          KHÔNG insert vô điều kiện (02-BUSINESS-RULES.md mục 0)
  5. internal/repository/location_repo.go — UpsertByUserAndLabel:
        - ST_MakePoint(lng, lat) / ST_SetSRID(..., 4326) → cột geom
  6. PostgreSQL/PostGIS               — bảng user_locations
[backend] → 200 JSON Location (hoặc error envelope)
```

**Request body**:

```json
{
  "label": "home",
  "latitude": 20.999375,
  "longitude": 105.811219,
  "city": "hanoi"
}
```

**Bảng field request**:

| Field | Kiểu | Bắt buộc | Ràng buộc / Mô tả |
| --- | --- | --- | --- |
| `label` | enum | ✅ | `current` \| `home` |
| `latitude` | number | ✅ | `[-90, 90]` |
| `longitude` | number | ✅ | `[-180, 180]` |
| `city` | string \| null | ⬜ | ≤ 100 ký tự; khuyến nghị `hanoi` \| `ho-chi-minh-city`. Nguồn: reverse-geocode phía FE hoặc null nếu không xác định |

**Response — 200 OK** (upsert; nếu `(user, label)` đã tồn tại → cập nhật tọa độ/city và trả row cũ với `created_at` giữ nguyên):

```json
{
  "id": "6f9d0e2c-4b5a-4c7d-9e1a-2b3c4d5e6f70",
  "label": "home",
  "latitude": 20.999375,
  "longitude": 105.811219,
  "city": "hanoi",
  "created_at": "2026-09-07T02:15:00Z"
}
```

**Lỗi**:

| HTTP | Code | Ví dụ điều kiện |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | `latitude = 95`, `label = "work"`, `city` > 100 ký tự, thiếu `longitude` |
| 401 | `UNAUTHORIZED` | Token không hợp lệ |

**Nhánh FE xử lý**: Flow 2 — GPS thành công / nhập tay thành công → nhận `Location` mới; sau đó có thể gọi `GET /me` để cập nhật trạng thái hoặc đi thẳng Dashboard.

---

### 4.4 GET /api/v1/aqi/current

**Mục đích**: trả AQI "hiện tại" tại 1 vị trí đã lưu của user (Dashboard). Vị trí được chọn bằng `location_id` hoặc mặc định là label `current`.

**Auth**: ✅ Bearer token.

**Chuỗi file tương tác**:

```
[frontend] lib/api/aqi.ts → (dashboard page / use-aqi.ts)
  → GET /api/v1/aqi/current?location_id=<uuid> (Bearer token)
[backend]
  1. cmd/api/main.go                    — route /api/v1/aqi/current, auth middleware
  2. internal/middleware/auth_middleware.go — verify JWT, gắn user_id
  3. internal/handler/aqi_handler.go — GetCurrent():
        - parse & validate location_id (format UUID) → VALIDATION_ERROR nếu sai
  4. internal/service/aqi_service.go — GetCurrent(userID, locationID?):
        a. xác định location: theo location_id (phải thuộc user) HOẶC mặc định label 'current'
           → location_repo.FindByIDAndUser / FindByUserAndLabel
           → không có → LOCATION_NOT_FOUND
        b. tìm trạm active gần nhất ≤ 20 km → station_repo.FindNearestWithin(lat, lng, 20000m)
           → không có trạm → NO_STATION_COVERAGE
        c. đọc cache Redis key `aqi:current:{station_id}` (database/redis.go):
           - hit → dùng reading đã cache (TTL 15–30 phút, < chu kỳ fetch 1h)
           - miss → aqi_reading_repo.GetLatest(station_id) → nếu có → ghi cache
        d. đánh giá dữ liệu:
           - chưa từng có reading nào → NO_DATA
           - reading.measured_at cũ hơn 3 giờ → DATA_STALE (không dùng số liệu cũ cho "hiện tại")
           - ngược lại → tính level + advice, trả 200
  5. internal/repository/
        location_repo.go / station_repo.go (PostGIS: ST_DWithin, trả kèm distance)
        aqi_reading_repo.go (TimescaleDB: lấy row mới nhất theo (station_id, time DESC))
  6. PostgreSQL (PostGIS + TimescaleDB) + Redis (cache-aside)
[backend] → 200 JSON CurrentAqi (hoặc error envelope)
```

**Request — query parameter**:

| Param | Bắt buộc | Mô tả |
| --- | --- | --- |
| `location_id` | ⬜ | UUID của vị trí user đã lưu. Bỏ trống → dùng location có `label = current`. Nếu user không có vị trí nào → `LOCATION_NOT_FOUND`. |

**Response — 200 OK**:

```json
{
  "location": {
    "id": "6f9d0e2c-4b5a-4c7d-9e1a-2b3c4d5e6f70",
    "label": "current",
    "latitude": 21.030578,
    "longitude": 105.854208,
    "city": "hanoi"
  },
  "station": {
    "id": "c4a9e5f1-2b7d-4f0a-9c3e-8d6b2a1f4e70",
    "name": "Cầu Giấy, Hà Nội",
    "city": "hanoi",
    "distance_m": 1240
  },
  "reading": {
    "aqi": 152,
    "level": "unhealthy",
    "measured_at": "2026-09-07T01:00:00Z",
    "dominant_pollutant": "pm25",
    "pollutants": {
      "pm25": 89.4,
      "pm10": null,
      "o3": null,
      "no2": null,
      "so2": null,
      "co": null
    }
  },
  "advice": {
    "summary": "Chất lượng không khí ở mức Xấu — bắt đầu ảnh hưởng tới toàn bộ người dân, nhóm nhạy cảm bị ảnh hưởng nghiêm trọng hơn.",
    "actions": [
      "Hạn chế ra ngoài khi không cần thiết",
      "Đeo khẩu trang N95 nếu phải ra đường",
      "Đóng cửa sổ và bật máy lọc không khí"
    ]
  }
}
```

**Bảng field**:

| Field | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| `location` | object | ✅ | Vị trí đã dùng để tra cứu (bỏ field `created_at`) |
| `location.id` / `label` / `latitude` / `longitude` / `city` | | ✅ | Như [3.1](#31-location-user_locations) |
| `station.id` | string (UUID) | ✅ | Station nội bộ (không phải WAQI id) |
| `station.name` | string | ✅ | Tên trạm hiển thị (VD: "Cầu Giấy, Hà Nội") |
| `station.city` | string \| null | ✅ | |
| `station.distance_m` | integer | ✅ | Khoảng cách từ location tới trạm (mét), luôn ≤ 20000 |
| `reading.aqi` | integer | ✅ | AQI tổng hợp |
| `reading.level` | enum | ✅ | Xem [2.8](#28-aqi-level-code) — để FE map màu/nhãn không cần tự phân loại lại |
| `reading.measured_at` | string (ISO) | ✅ | Thời điểm đo (cột `time` trong `aqi_readings`) |
| `reading.dominant_pollutant` | string \| null | ✅ | `pm25` \| `pm10` \| `o3` \| `no2` \| `so2` \| `co` \| null |
| `reading.pollutants` | object | ✅ | Luôn đủ 6 key; thiếu chỉ số phụ → `null` (không ảnh hưởng khả năng cảnh báo — `02-BUSINESS-RULES.md` mục 3) |
| `advice.summary` | string | ✅ | Khuyến nghị tổng hợp (backend sinh theo mức AQI — `06-SYSTEM-FLOW.md` GĐ5) |
| `advice.actions` | array[string] | ✅ | Danh sách hành động gợi ý |

**Quy tắc "dữ liệu có thể chưa cập nhật" (2 giờ)** — xử lý ở FE:

- Backend chỉ chặn dữ liệu cũ **> 3 giờ** (`DATA_STALE`).
- Với dữ liệu **2–3 giờ** tuổi, API vẫn trả `200` + `reading.measured_at`; FE tự so sánh `measured_at` với giờ hiện tại, nếu ≥ 2 giờ hiển thị nhãn **"Dữ liệu có thể chưa cập nhật"** (`02-BUSINESS-RULES.md` mục 3 — ngưỡng UI 2h thấp hơn ngưỡng stale 3h để báo sớm cho user).

**Lỗi**:

| HTTP | Code | Ví dụ điều kiện |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | `location_id` không đúng format UUID |
| 401 | `UNAUTHORIZED` | Token không hợp lệ |
| 404 | `LOCATION_NOT_FOUND` | `location_id` không tồn tại / không thuộc user; hoặc user không có location `current` khi không truyền `location_id` |
| 404 | `NO_STATION_COVERAGE` | Không có trạm active trong 20 km → FE hiển thị "chưa có dữ liệu khu vực này" |
| 404 | `NO_DATA` | Trạm hợp lệ nhưng chưa từng có reading (worker chưa chạy) → FE hiển thị trạng thái "chưa có dữ liệu" |
| 422 | `DATA_STALE` | Reading mới nhất cũ hơn 3 giờ → FE hiển thị "dữ liệu chưa cập nhật / thử lại sau" |

---

## 5. Đối chiếu với USER-FLOWS (7 flow)

Đảm bảo **mọi nhánh rẽ trong USER-FLOWS đều có response field / error code tương ứng** để FE xử lý. Cột "Field/Code FE dùng" ghi đúng dữ liệu FE cần đọc để rẽ nhánh.

| Flow | Nhánh rẽ | Endpoint phục vụ | Field/Code FE dùng | Ghi chú |
| --- | --- | --- | --- | --- |
| **1. Đăng ký/Đăng nhập** | Đăng nhập Keycloak thất bại | — (Keycloak/NextAuth, ngoài backend) | — | Xử lý hoàn toàn ở FE |
| | Lần đầu đăng nhập (chưa có `keycloak_id`) | `GET /me` | backend upsert user tự động; `health_profile == null` | FE → Onboarding |
| | Đã từng đăng nhập | `GET /me` | `health_profile != null` | FE → Dashboard |
| **2. Onboarding** | Đã có `health_profile` (quay lại app) | `GET /me` | `health_profile != null` → bỏ qua Onboarding | |
| | Tạo hồ sơ sức khỏe | `POST /health-profile` | `200` + `alert_threshold.threshold_aqi`, `is_custom` | |
| | Nhập hồ sơ không hợp lệ | `POST /health-profile` | `400 VALIDATION_ERROR` / `422 THRESHOLD_TOO_LOW` | FE hiện lỗi dưới field |
| | Từ chối GPS → nhập địa chỉ tay | `POST /locations` | `200` + `Location` | Nhập `label: "home"` |
| | Bỏ qua cả 2 cách nhập vị trí | `GET /me` | `locations: []` | Vào Dashboard hạn chế + CTA bổ sung vị trí |
| **3. Tùy chỉnh ngưỡng** | Giá trị không hợp lệ (< 50) | `POST /health-profile` | `422 THRESHOLD_TOO_LOW` | FE giữ nguyên giá trị cũ |
| | Giá trị không hợp lệ khác (> 500 / sai kiểu) | `POST /health-profile` | `400 VALIDATION_ERROR` (+ `details[].field`) | FE giữ nguyên giá trị cũ |
| | Lưu thành công | `POST /health-profile` | `200` + `alert_threshold.is_custom == true` | |
| **4. Web Push** | Trình duyệt không hỗ trợ Push API / từ chối Notification | — (endpoint push ở Bước 7/8, `POST /api/v1/push-subscriptions`) | — | Ngoài phạm vi 4 endpoint của slice; nhánh lỗi là FE-only |
| **5. Dashboard** | Chưa đăng nhập | `GET /me`, `GET /aqi/current` | `401 UNAUTHORIZED` | FE redirect login |
| | Chưa có vị trí đã lưu | `GET /me` | `locations: []` | FE hiện CTA Onboarding (không gọi `/aqi/current`) |
| | Cache Redis miss + TimescaleDB chưa có dữ liệu | `GET /aqi/current` | `404 NO_DATA` | FE trạng thái "chưa có dữ liệu" |
| | Reading 2–3 giờ tuổi | `GET /aqi/current` | `200` + `reading.measured_at` (≥ 2h so với now) | FE nhãn "Dữ liệu có thể chưa cập nhật" |
| | Reading quá cũ (> 3 giờ) | `GET /aqi/current` | `422 DATA_STALE` | FE trạng thái "chưa có dữ liệu mới" |
| | Không có trạm trong 20 km | `GET /aqi/current` | `404 NO_STATION_COVERAGE` | FE "chưa có dữ liệu khu vực này" |
| | Có dữ liệu đầy đủ | `GET /aqi/current` | `200` + `reading.aqi`, `reading.level`, `advice` | FE vẽ gauge/badge/map/khuyến nghị |
| **6. History** | Không có dữ liệu trong khoảng đã chọn | — (endpoint `GET /api/v1/aqi/history` ở Bước 7/8) | — | Ngoài phạm vi 4 endpoint của slice |
| **7. Alerts** | Danh sách rỗng | — (endpoint `GET /api/v1/alerts` ở Bước 7/8) | — | Ngoài phạm vi 4 endpoint của slice |

> Kết luận đối chiếu: các nhánh của Flow 1, 2, 3, 5 (tương ứng 4 endpoint) đều có field/code phản hồi đầy đủ. Flow 4, 6, 7 nằm ngoài slice và cần endpoint bổ sung (xem mục 6) — đã được nhận diện tường minh, không bỏ sót âm thầm.

---

## 6. Quyết định & backlog sau Bước 6

**Quyết định đã chốt (thuộc Bước 6):**

1. Versioning theo path `/api/v1`; success trả resource trực tiếp; lỗi luôn dùng envelope `{error: {code, message, details?}}` ([2.9](#29-format-response-thành-công), [2.10](#210-format-response-lỗi)).
2. ID = UUID v4 string; timestamp = ISO 8601/RFC 3339 UTC (`Z`).
3. `POST /health-profile` và `POST /locations` là **upsert idempotent**, trả `200 OK` + trạng thái cuối (không `201`).
4. Ngưỡng cảnh báo được đổi qua `POST /health-profile` (gửi lại full payload kèm `custom_threshold_aqi`) vì slice chưa có endpoint `PATCH` riêng; FE phải gửi kèm `custom_threshold_aqi` nếu muốn **giữ** ngưỡng custom khi sửa hồ sơ (không gửi → tính lại mặc định).
5. `/aqi/current` chặn dữ liệu cũ **> 3h** bằng `DATA_STALE`; nhãn "có thể chưa cập nhật" cho dữ liệu **2–3h** do FE tự suy từ `reading.measured_at`.
6. `PROFILE_NOT_FOUND` được định nghĩa trong taxonomy nhưng **không endpoint nào trong 4 endpoint này trả** — dành cho các endpoint profile/threshold riêng ở Bước 7/8 (sẽ bổ sung khi mở rộng contract).

**Backlog / việc phải làm ở Bước 7–8:**

- Implement 4 endpoint đúng chuỗi file đã liệt kê + Ingestion Worker tối thiểu.
- Xác nhận Keycloak Realm/Client trước khi code `auth_middleware.go`.
- Chạy thử `migrate up` đủ 11 migration, verify bằng `\dt` trong `psql`.
- Mở rộng contract khi thêm endpoint: `POST /api/v1/push-subscriptions` (Flow 4), `GET /api/v1/aqi/history` (Flow 6), `GET /api/v1/alerts` (Flow 7), và các endpoint profile/threshold (nơi dùng `PROFILE_NOT_FOUND`).
