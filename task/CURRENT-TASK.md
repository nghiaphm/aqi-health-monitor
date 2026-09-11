# Current Task

> File này chỉ chứa checklist của **bước đang làm**. Khi bước này hoàn thành, **xóa sạch nội dung và viết lại cho bước kế tiếp**.

## Bước 7 — Vertical Slice (Tracer Bullet)

**Mục 0 (blocking)**: đã chốt tạm thời bởi bạn, sẽ update chi tiết sau — không chặn tiến độ.

### ✅ Đã xong — Mục 1: Backend — Auth & User Sync (`GET /me`)

6 commit, nhánh `feature/auth-user-sync`, Definition of Done pass đủ 4/4. Chi tiết xem `task/CHANGELOG.md`.

### ✅ Đã xong — Mục 2: Backend — Health Profile (`POST /health-profile`)

Definition of Done pass đủ 5/5 (unit test đủ 18/18 tổ hợp ma trận + 3 tổ hợp test thật qua HTTP; upsert-in-place trong 1 transaction; `custom_threshold_aqi=49` → `422 THRESHOLD_TOO_LOW`; enum sai → `400 VALIDATION_ERROR`; response khớp 100% mục 4.2). Chi tiết xem `task/CHANGELOG.md`.

### ⬜ Chờ duyệt — Mục 3: Backend — Locations (`POST /locations`)

- [ ] `internal/handler/location_handler.go` — `Upsert()`: validate `label ∈ {current, home}`, `latitude ∈ [-90,90]`, `longitude ∈ [-180,180]`, `city` = null hoặc string ≤ 100 ký tự; sai → `400 VALIDATION_ERROR` kèm `details[].field`; trả shape mục 4.3
- [ ] `internal/service/location_service.go` — `UpsertByUserAndLabel`: lấy user nội bộ qua `UserSyncer.FindOrCreate`; enforce invariant "1 user tối đa 1 vị trí mỗi label" — upsert theo `(user_id, label)`, KHÔNG insert vô điều kiện (`BUSINESS-RULES.md` mục 0)
- [ ] `internal/repository/location_repo.go` — thêm `UpsertByUserAndLabel` (hàm ghi, khác `FindAllByUserID` đọc ở Mục 1): `ST_MakePoint(lng, lat)` + `ST_SetSRID(..., 4326)` → cột `geom`
- [ ] Đăng ký route `POST /api/v1/locations` trong `cmd/api/main.go`, gắn `auth_middleware`

### Definition of Done — Mục 3

- [ ] Gọi `POST /api/v1/locations` không có token → `401 UNAUTHORIZED`
- [ ] JWT hợp lệ, user chưa có vị trí → tạo mới, trả đúng shape mục 4.3 (`id`, `label`, `latitude`, `longitude`, `city`, `created_at`)
- [ ] Gọi lại lần 2 cùng `label` với tọa độ khác → update-in-place, **không tạo row mới** (đếm DB = 1), giữ nguyên `created_at`
- [ ] User có 2 `label` khác nhau (`current` + `home`) → 2 row độc lập (đúng invariant)
- [ ] `latitude = 95` hoặc `label = "work"` → `400 VALIDATION_ERROR`
- [ ] `city` > 100 ký tự → `400 VALIDATION_ERROR`
- [ ] Response field khớp 100% bảng field mục 4.3 `API-CONTRACT.md` — không thừa/thiếu field

> **Chưa code Mục 3** — chờ xác nhận checklist này trước khi bắt đầu.

### Backlog kế tiếp trong Bước 7

- Mục 4 — Ingestion Worker tối thiểu (chạy tay `sync_stations` + `fetch_readings` 1 lần)
- Mục 5 — AQI Current (`GET /aqi/current`)
- Mục 6 — Frontend nối dây tối thiểu (thêm UI `sensitivity_level` khi code Onboarding)
