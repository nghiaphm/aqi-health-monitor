# System Flow — AQI Health Monitor

> **Vai trò của tài liệu này**: mô tả toàn bộ vòng đời vận hành hệ thống theo trình tự thời gian, dưới dạng văn xuôi dễ đọc — dùng để giới thiệu nhanh "hệ thống hoạt động thế nào" cho người đọc lần đầu (nhà tuyển dụng, cộng tác viên mới).
>
> Đây **không phải User Flow** thuần túy — tài liệu bao gồm cả phần người dùng chủ động thao tác lẫn phần hệ thống tự vận hành nền, vì cả hai đều cần thiết để hiểu đúng bản chất sản phẩm (cảnh báo AQI hoạt động ngay cả khi user không mở app).
>
> Xem `ARCHITECTURE.md` mục 4 để có **sơ đồ kỹ thuật formal** (C4, sequence diagram) cho từng luồng bên dưới. Khi có thay đổi logic nghiệp vụ, cập nhật `ARCHITECTURE.md` trước, tài liệu này chỉ diễn giải lại theo mạch thời gian.

## Phân loại các giai đoạn

| Giai đoạn | Loại | Người/hệ thống khởi xướng |
| --- | --- | --- |
| 0 — Khởi động hệ thống | 🔧 Background | Hệ thống (không có user) |
| 1 — Xác thực lần đầu | 👤 User-driven | User |
| 2 — Onboarding | 👤 User-driven | User |
| 3 — Đăng ký Web Push | 👤 User-driven | User (tùy chọn) |
| 4 — Ingestion & Alert | 🔧 Background | Hệ thống (lặp mỗi giờ) |
| 5 — Xem Dashboard/History/Alerts | 👤 User-driven | User (lặp lại, bất kỳ lúc nào) |

---

## 🔧 Giai đoạn 0 — Hệ thống khởi động

*Background — không có user tham gia.*

Ngay khi `docker-compose up`, worker bắt đầu chạy độc lập, không chờ có user nào truy cập:

1. Asynq scheduler kích hoạt job `sync_stations` — gọi WAQI `/map/bounds/` cho khu vực Hà Nội + TP.HCM → upsert vào bảng `stations`.
2. Ngay sau đó, job `fetch_readings` chạy theo lịch (mỗi giờ) — duyệt qua từng `stations` đang active, gọi WAQI `/feed/@id/`, lưu kết quả vào `aqi_readings` (TimescaleDB) và cập nhật Redis cache.

**Ý nghĩa**: dữ liệu AQI đã sẵn sàng trong hệ thống **trước khi** user đầu tiên đăng ký. Đây là lý do Ingestion Worker được ưu tiên code trước — mọi module khác đều phụ thuộc vào dữ liệu này tồn tại sẵn.

---

## 👤 Giai đoạn 1 — Xác thực lần đầu (Authentication)

*User-driven.*

1. User mở Frontend (Next.js) → bấm "Đăng nhập" → NextAuth redirect sang Keycloak hosted login page.
2. User đăng ký/đăng nhập tại Keycloak → Keycloak trả về JWT (`access_token`) → Frontend lưu session.
3. Frontend gọi API backend lần đầu (VD: `GET /me`) kèm `Authorization: Bearer <JWT>`.
4. Backend verify JWT qua JWKS → tìm `keycloak_id` trong bảng `users` → chưa có → tự động tạo record mới (upsert) → trả về `user_id` nội bộ.

Từ thời điểm này, mọi API call tiếp theo của user đều gắn với `user_id` này.

---

## 👤 Giai đoạn 2 — Onboarding (thiết lập hồ sơ)

*User-driven — chỉ xảy ra 1 lần cho mỗi user.*

1. Frontend hiển thị form: chọn bệnh lý (`none` / `asthma` / `allergic_rhinitis`), nhóm tuổi (`child` / `adult` / `elderly`).
2. Submit → API tạo record trong `health_profiles`.
3. Backend tự tính `threshold_aqi` mặc định dựa trên `condition_type` + `age_group` (theo bảng ngưỡng y tế đã research) → lưu vào `alert_thresholds` (`is_custom = false`).
4. User có thể tùy chỉnh ngưỡng thủ công sau đó → update `alert_thresholds`, đánh dấu `is_custom = true`.
5. User cấp quyền GPS (hoặc nhập địa chỉ nhà) → Frontend gửi tọa độ → API lưu vào `user_locations` (cột `geom` kiểu PostGIS).

---

## 👤 Giai đoạn 3 — Đăng ký nhận Web Push

*User-driven — tùy chọn, không bắt buộc.*

1. User bấm "Bật thông báo" trên trình duyệt.
2. Frontend đăng ký Service Worker + Push Manager của trình duyệt → nhận về `endpoint`, `p256dh`, `auth` key.
3. Gửi lên backend → lưu vào `push_subscriptions`.

Sau bước này, user đã "sẵn sàng" nhận cảnh báo — nhưng cảnh báo chỉ thực sự được gửi khi Giai đoạn 4 (nền) phát hiện AQI vượt ngưỡng.

---

## 🔧 Giai đoạn 4 — Vòng lặp nền: Ingestion → Alert

*Background — lặp lại mỗi giờ, độc lập hoàn toàn với việc user có đang mở app hay không.*

Đây là "trái tim" của hệ thống:

1. **Fetch**: worker lấy AQI mới nhất cho từng trạm → lưu `aqi_readings`.
2. **Match**: ngay sau khi 1 trạm có dữ liệu mới, worker enqueue job `check_alerts(station_id)` → query `user_locations` xem ai đang ở gần trạm này (PostGIS) → với mỗi user, lấy `alert_thresholds` tương ứng.
3. **So sánh**: AQI mới có vượt `threshold_aqi` của user đó không?
   - **Không** → dừng lại, không làm gì thêm.
   - **Có** → kiểm tra `alerts_log`: đã gửi cảnh báo cho user này trong X giờ gần đây chưa?
     - Đã gửi rồi → bỏ qua (chống spam).
     - Chưa gửi → tiếp tục.
4. **Gửi cảnh báo**: worker lấy `push_subscriptions` của user → gửi Web Push; đồng thời gửi Email qua Resend.
5. **Ghi log**: lưu kết quả (`sent` / `failed`) vào `alerts_log`.

User nhận được thông báo trên trình duyệt hoặc email mà **không cần mở app** — đây là giá trị cốt lõi "chủ động cảnh báo" đã đặt ra từ `RESEARCH.md`.

---

## 👤 Giai đoạn 5 — Xem Dashboard / History / Alerts

*User-driven — lặp lại bất kỳ lúc nào, độc lập với Giai đoạn 4.*

1. User mở app → Frontend gọi API với vị trí đã lưu (`user_locations`).
2. Backend tìm trạm gần nhất (PostGIS `ST_Distance`).
3. Check Redis cache theo `station_id` → hit thì trả ngay; miss thì query `aqi_readings` (TimescaleDB) rồi cache lại.
4. Trả về: AQI hiện tại, mức cảnh báo, khuyến nghị hành động → Frontend hiển thị bản đồ + gauge.
5. Nếu user mở trang **History**: API query `aqi_readings` theo khoảng thời gian (dùng `time_bucket` để gộp theo giờ/ngày) → vẽ biểu đồ.
6. Nếu user mở trang **Alerts**: query `alerts_log` theo `user_id` → hiển thị lịch sử đã nhận.

---

## Tổng quan mối quan hệ giữa 2 loại luồng

```
🔧 Luồng NỀN (Giai đoạn 0, 4 — Worker, tự động, lặp mỗi giờ)
   Fetch AQI → Lưu DB → So khớp ngưỡng → Gửi cảnh báo
   (chạy bất kể user có online hay không)

👤 Luồng USER (Giai đoạn 1, 2, 3, 5 — API, theo yêu cầu)
   Đăng nhập → Onboarding (1 lần) → Bật Push (tùy chọn) → Xem Dashboard/History/Alerts (nhiều lần)
   (đọc dữ liệu mà luồng Nền đã chuẩn bị sẵn)

→ Điểm giao nhau duy nhất: dữ liệu trong Postgres/PostGIS/TimescaleDB + Redis
```

Đây cũng là lý do thứ tự triển khai tính năng ưu tiên **Ingestion Worker (Giai đoạn 0, 4)** trước — vì Giai đoạn 5 (Dashboard) không có gì để hiển thị nếu chưa có dữ liệu do luồng nền tạo ra.
