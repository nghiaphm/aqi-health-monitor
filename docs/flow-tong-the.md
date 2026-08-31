# Flow tổng thể

## 1. Research & Requirements

- Đọc kỹ docs API WAQI (aqicn.org/api/) — giới hạn rate limit, format response, có bao nhiêu trạm ở VN (Hà Nội, TP.HCM, Đà Nẵng...).
- Xác định phạm vi MVP: bao nhiêu thành phố/khu vực hỗ trợ ban đầu? (nên giới hạn 2-3 thành phố lớn để không tốn công fetch/test quá rộng).
- Tìm hiểu ngưỡng AQI y tế thực tế (WHO, EPA) để làm cơ sở logic cảnh báo — ví dụ AQI > 150 nguy hiểm cho người hen suyễn, > 200 nguy hiểm cho người bình thường.
- Xác định input "lịch sử bệnh lý cá nhân" sẽ đơn giản đến đâu (MVP: chọn bệnh lý từ danh sách có sẵn — hen suyễn, dị ứng phấn hoa, viêm mũi... + mức độ nhạy cảm).

## 2. Data Modeling

- Schema chính: users, health_profiles, locations (lưu tọa độ + trạm AQI gần nhất), aqi_readings (time-series), alerts_sent (log lịch sử cảnh báo, tránh spam).
- Thiết kế bảng aqi_readings theo hướng tối ưu cho TimescaleDB (hypertable theo thời gian).
- Thiết kế ngưỡng cảnh báo cá nhân hóa: là 1 bảng riêng alert_thresholds (user_id, pollutant_type, threshold_value) để linh hoạt, không hardcode logic.

## 3. Backend Core — API & Data Ingestion

- Setup Go project (cấu trúc thư mục theo chuẩn cmd, internal, pkg).
- Viết module gọi WAQI API, parse response, xử lý lỗi/rate limit.
- Viết cron job (Asynq periodic task) fetch AQI định kỳ (mỗi giờ) cho các trạm đã đăng ký, lưu vào TimescaleDB.
- Test cron job chạy độc lập trước khi tích hợp API user-facing (tách rời phần "ingest" và phần "serve" ngay từ đầu — dễ debug hơn).

## 4. Geospatial Layer

- Setup PostGIS, viết query tìm trạm AQI gần user nhất theo tọa độ (ST_Distance).
- API endpoint: user nhập địa chỉ hoặc bật GPS → convert sang lat/long (dùng geocoding API như Nominatim/OpenStreetMap, miễn phí) → trả về AQI khu vực gần nhất.

## 5. Personalization & Alert Logic

- API cho user cấu hình hồ sơ sức khỏe + ngưỡng cảnh báo cá nhân.
- Worker (Asynq) chạy sau mỗi lần fetch AQI mới: so sánh với ngưỡng của từng user trong khu vực đó → nếu vượt ngưỡng → queue job gửi notification.
- Logic tránh spam: không gửi lại cảnh báo nếu đã gửi trong X giờ gần đây cho cùng mức độ.

## 6. Notification System

- Tích hợp Resend/Mailgun gửi email cảnh báo.
- Template email đơn giản: AQI hiện tại, mức độ nguy hiểm, khuyến nghị hành động (VD: "hạn chế ra ngoài, đeo khẩu trang N95").

## 7. API Design hoàn chỉnh

- Tổng hợp toàn bộ REST API: auth, health profile, location, AQI hiện tại/lịch sử, cấu hình ngưỡng, lịch sử cảnh báo.
- Viết API docs (Swagger/OpenAPI) — vừa để tự quản lý, vừa để tích hợp frontend dễ hơn.

## 8. Frontend

- Trang đăng ký/cấu hình hồ sơ sức khỏe.
Dashboard chính: bản đồ AQI (Leaflet) + AQI khu vực hiện tại của user.
- Biểu đồ lịch sử AQI (Recharts) theo ngày/tuần.
- Trang lịch sử cảnh báo đã nhận.

## 9. DevOps — Containerization & Local Orchestration

- Viết Dockerfile cho Go backend (multi-stage build), Next.js frontend.
- Docker Compose: backend + frontend + Postgres/TimescaleDB + Redis chạy đồng bộ local.
- Đảm bảo .env quản lý config sạch sẽ (API key WAQI, DB credentials, email API key).

## 10. Testing

- Unit test cho logic tính ngưỡng cảnh báo (đây là business logic quan trọng nhất, cần test kỹ).
- Integration test cho luồng: fetch AQI → lưu DB → trigger alert.
- Test cron job với mock data (không gọi API thật liên tục khi test).

## 11. CI/CD

- GitHub Actions: chạy test tự động khi push/PR.
- Build Docker image tự động, push lên registry (Docker Hub hoặc GitHub Container Registry).
- Auto-deploy khi merge vào main (Railway/Fly.io hỗ trợ deploy từ Git trực tiếp, hoặc tự viết deploy script cho VPS).

## 12. Deploy thật

- Deploy lên Railway/Fly.io hoặc VPS.
Setup domain + HTTPS (Let's Encrypt nếu dùng VPS).
- Đảm bảo cron job vẫn chạy đúng trên môi trường production (kiểm tra timezone!).

## 13. Monitoring & Observability

- Prometheus scrape metrics từ Go backend (số lần fetch AQI thành công/thất bại, số alert đã gửi).
- Grafana dashboard: uptime cron job, latency API, tỷ lệ lỗi khi gọi WAQI.
- Log tập trung (structured logging + Loki nếu muốn nâng cao).

## 14. Polish & Documentation

- Viết README rõ ràng: kiến trúc hệ thống, cách chạy local, cách deploy.
- Vẽ sơ đồ kiến trúc tổng thể (đây là phần quan trọng để show trong portfolio/CV).
