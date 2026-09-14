# Current Task

> File này chỉ chứa checklist của **bước đang làm**. Khi bước này hoàn thành, **xóa sạch nội dung và viết lại cho bước kế tiếp**.

## Bước 7 — Vertical Slice (Tracer Bullet)

**Mục 0 (blocking)**: đã chốt tạm thời bởi bạn, sẽ update chi tiết sau — không chặn tiến độ.

### ✅ Đã xong — Mục 1: Backend — Auth & User Sync (`GET /me`)

6 commit, nhánh `feature/auth-user-sync`, Definition of Done pass đủ 4/4. Chi tiết xem `task/CHANGELOG.md`.

### ✅ Đã xong — Mục 2: Backend — Health Profile (`POST /health-profile`)

Definition of Done pass đủ 5/5 (unit test đủ 18/18 tổ hợp ma trận + 3 tổ hợp test thật qua HTTP; upsert-in-place trong 1 transaction; `custom_threshold_aqi=49` → `422 THRESHOLD_TOO_LOW`; enum sai → `400 VALIDATION_ERROR`; response khớp 100% mục 4.2). Chi tiết xem `task/CHANGELOG.md`.

### ✅ Đã xong — Mục 3: Backend — Locations (`POST /locations`)

Migration `000012` thêm `UNIQUE (user_id, label)` → upsert atomic bằng `ON CONFLICT DO UPDATE`; Definition of Done pass đủ 7/7 (tạo mới + shape mục 4.3; update-in-place giữ nguyên `created_at` verify trực tiếp DB; 2 label độc lập; validate → `400`; 401 khi thiếu token). Chi tiết xem `task/CHANGELOG.md`.

### ✅ Đã xong — Mục 4: Ingestion Worker tối thiểu (`sync_stations` + `fetch_readings`)

`pkg/waqi` client + 2 Asynq task + `aqi_service`; reading + `last_synced_at` gộp **cùng 1 transaction** (phương án b), `ON CONFLICT (station_id, time) DO NOTHING`; Redis cache `aqi:current:{station_id}` TTL 20 phút. Definition of Done pass đủ 9/9 (sync idempotent; bỏ qua record thiếu `aqi` với log WARNING không fail job; `last_synced_at` + Redis verify trực tiếp; chạy lại không tạo trùng; không đụng Alert Evaluation). Chi tiết xem `task/CHANGELOG.md`.

### ⬜ Chờ duyệt — Mục 5: Backend — AQI Current (`GET /aqi/current`)

- [ ] `internal/handler/aqi_handler.go` — `GetCurrent()`: parse & validate `location_id` (query, optional, đúng format UUID → sai thì `400 VALIDATION_ERROR`); trả đúng shape mục 4.4
- [ ] `internal/service/aqi_service.go` — mở rộng thêm `GetCurrent(ctx, claims, locationID *string)`:
  - resolve location: theo `location_id` (phải thuộc user) HOẶC mặc định location có `label = current`; không có → `404 LOCATION_NOT_FOUND`
  - tìm trạm active gần nhất **≤ 20 km** (`BUSINESS-RULES.md` mục 3); không có → `404 NO_STATION_COVERAGE`
  - cache-aside Redis key `aqi:current:{station_id}`: hit → dùng; miss → `GetLatest` rồi ghi lại cache; DB chưa từng có reading → `404 NO_DATA`
  - **rule `DATA_STALE`**: reading cũ hơn **3 giờ** so với hiện tại → `422 DATA_STALE` (không trả số liệu cũ như "hiện tại")
  - tính `level` theo bảng 6 mức ở `API-CONTRACT.md` mục 2.8 và `advice` (summary + actions) theo bảng khuyến nghị `RESEARCH.md` mục 2.2
- [ ] `internal/repository/station_repo.go` — thêm `FindNearestWithin(ctx, lat, lng, maxMeters)` dùng PostGIS (`ST_DWithin` + `ST_Distance`, order by distance, limit 1), trả kèm `distance_m`
- [ ] `internal/repository/aqi_reading_repo.go` — thêm `GetLatest(ctx, stationID)` (`ORDER BY time DESC LIMIT 1`)
- [ ] `internal/repository/location_repo.go` — thêm bản đọc `FindByIDAndUserID` + `FindByUserAndLabel` để resolve location
- [ ] Đăng ký route `GET /api/v1/aqi/current` trong `cmd/api/main.go`, gắn `auth_middleware`

### Definition of Done — Mục 5

- [ ] `GET /api/v1/aqi/current` không token → `401 UNAUTHORIZED`
- [ ] User chưa có vị trí nào (không truyền `location_id`) → `404 LOCATION_NOT_FOUND`
- [ ] `location_id` không thuộc user → `404 LOCATION_NOT_FOUND`
- [ ] Có trạm trong 20 km + reading mới → `200`, shape đúng mục 4.4: `location`/`station`(có `distance_m` ≤ 20000)/`reading`/`advice`; `pollutants` đủ 6 key
- [ ] Reading cũ hơn 3 giờ → `422 DATA_STALE` (test bằng cách seed reading có `time` cũ)
- [ ] Không có trạm trong 20 km → `404 NO_STATION_COVERAGE`
- [ ] Trạm có nhưng chưa có reading (DB rỗng + cache miss) → `404 NO_DATA`
- [ ] `level` + `advice` khớp mức AQI theo bảng mục 2.8 (test ít nhất 2 mức khác nhau, VD `unhealthy` và `good`)
- [ ] Redis cache-hit: gọi lần 2 cùng trạm → vẫn `200`, dữ liệu lấy từ cache (verify `TTL`/`GET` bằng `redis-cli`, không chỉ tin log)
- [ ] Response field-set khớp 100% bảng field mục 4.4 `API-CONTRACT.md` — không thừa/thiếu

> **Chưa code Mục 5** — chờ xác nhận checklist này trước khi bắt đầu.
> Phụ thuộc: dữ liệu test có thể seed trực tiếp vào `stations`/`aqi_readings`/`user_locations` (chưa có `WAQI_API_TOKEN` thật).

### Backlog kế tiếp trong Bước 7

- Mục 6 — Frontend nối dây tối thiểu (thêm UI `sensitivity_level` khi code Onboarding)
