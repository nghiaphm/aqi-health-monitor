# Business Rules — AQI Health Monitor

> Tài liệu này chốt các **quyết định nghiệp vụ cụ thể** (con số, công thức, invariant) mà `RESEARCH.md`, `ARCHITECTURE.md` và `DATABASE_DESIGN.md` chưa quy định rõ. Đây là nguồn tham chiếu bắt buộc khi code Ingestion Worker và Alert Evaluation — không tự quyết định số liệu ngay trong code mà không đối chiếu file này trước.
>
> Khi 1 rule thay đổi trong quá trình phát triển, cập nhật ở đây trước, sau đó mới sửa code — giữ đúng nguyên tắc đã áp dụng xuyên suốt dự án (tài liệu dẫn dắt code, không phải ngược lại).

## 0. Domain Invariants (bất biến nghiệp vụ)

Các điều kiện sau **phải luôn đúng**, nhưng không được ràng buộc cứng ở tầng DB (do giữ schema đơn giản) — **bắt buộc enforce ở tầng Service**:

| Invariant | Enforce ở đâu | Ghi chú |
| --- | --- | --- |
| 1 user có tối đa 1 `HealthProfile` | DB (UNIQUE `user_id`) | Đã ràng buộc cứng ở DB |
| 1 user có tối đa 1 vị trí cho mỗi `label` (`current`/`home`) | **Service layer** | DB không có UNIQUE `(user_id, label)` — service phải dùng upsert theo `(user_id, label)`, không insert vô điều kiện |
| 1 user có đúng 1 `AlertThreshold` đang hiệu lực tại 1 thời điểm | **Service layer** | DB không ràng buộc UNIQUE `user_id` trên `alert_thresholds` (cho phép mở rộng lịch sử threshold sau này) — nhưng ở MVP, service phải update-in-place (upsert), không tạo thêm row mới mỗi lần user đổi ngưỡng |
| `AlertLog` không được tạo nếu user chưa có `AlertThreshold` | **Service layer** | Alert Evaluation phải skip user nếu chưa có threshold, không dùng giá trị mặc định ngầm |

## 1. Ngưỡng cảnh báo mặc định (Default Threshold Matrix)

### Công thức

```
threshold_aqi = base(condition_type) + age_modifier(age_group) + sensitivity_modifier(sensitivity_level)
clamp(threshold_aqi, min=50, max=300)
```

### Bảng tra cứu

| condition_type | Base |
| --- | --- |
| `none` | 150 |
| `asthma` | 100 |
| `allergic_rhinitis` | 100 |

| age_group | Modifier |
| --- | --- |
| `child` | −20 |
| `adult` | 0 |
| `elderly` | −20 |

| sensitivity_level | Modifier |
| --- | --- |
| `normal` | 0 |
| `high` | −20 |

### Ma trận đầy đủ (18 tổ hợp — dùng trực tiếp làm test case khi code)

| condition_type | age_group | sensitivity_level | threshold_aqi |
| --- | --- | --- | --- |
| none | child | normal | 130 |
| none | child | high | 110 |
| none | adult | normal | 150 |
| none | adult | high | 130 |
| none | elderly | normal | 130 |
| none | elderly | high | 110 |
| asthma | child | normal | 80 |
| asthma | child | high | 60 |
| asthma | adult | normal | 100 |
| asthma | adult | high | 80 |
| asthma | elderly | normal | 80 |
| asthma | elderly | high | 60 |
| allergic_rhinitis | child | normal | 80 |
| allergic_rhinitis | child | high | 60 |
| allergic_rhinitis | adult | normal | 100 |
| allergic_rhinitis | adult | high | 80 |
| allergic_rhinitis | elderly | normal | 80 |
| allergic_rhinitis | elderly | high | 60 |

### Ràng buộc khi user tự chỉnh threshold thủ công (`is_custom = true`)

- DB đã CHECK constraint `0 ≤ threshold_aqi ≤ 500`.
- **Bổ sung validation ở Service layer**: từ chối giá trị **dưới 50** — dưới ngưỡng "Trung bình" (50) thì cảnh báo liên tục mất ý nghĩa (gần như AQI ngày nào cũng vượt). Trả lỗi rõ ràng cho Frontend hiển thị, không âm thầm clamp lại giá trị.
- Không giới hạn trần ngoài CHECK 500 — user có quyền tự đặt ngưỡng cao (ít cảnh báo hơn) nếu muốn.

### Vì sao không lưu ma trận này trong DB

Đã cân nhắc bảng cấu hình riêng, nhưng quyết định giữ dưới dạng constant trong code (`internal/service/health_profile_service.go`) — 18 tổ hợp cố định, hiếm thay đổi, tạo bảng DB riêng là dư thừa (xem ghi chú tương ứng trong `DATABASE_DESIGN.md`).

## 2. Chống spam & tần suất cảnh báo

| Rule | Giá trị đã chốt |
| --- | --- |
| **Cooldown period** | **6 giờ** — không gửi lại cảnh báo cho cùng 1 cặp `(user, station)` trong vòng 6 giờ kể từ lần gửi gần nhất |
| **Dedup key** | `(user_id, station_id)` — **không phải** chỉ `user_id` |
| **Lý do dùng `station_id` làm 1 phần key** | User có thể có 2 vị trí (`current` + `home`) map tới 2 trạm khác nhau — đây là 2 tình huống ô nhiễm thực sự khác nhau, không nên chặn lẫn nhau. Nếu 2 vị trí trùng cùng 1 trạm, dedup tự nhiên gộp lại (đúng ý muốn, tránh gửi 2 email giống hệt nhau) |
| **Query thực thi** | `SELECT 1 FROM alerts_log WHERE user_id = ? AND station_id = ? AND sent_at > now() - INTERVAL '6 hours' AND status = 'sent'` — dùng index `idx_alerts_log_user_station_sent_at` đã tạo ở migration `000010` |

## 3. Chọn trạm & chất lượng dữ liệu đầu vào

| Rule | Giá trị đã chốt | Lý do |
| --- | --- | --- |
| **Khoảng cách hợp lệ tối đa** | **20 km** | Trạm cách quá xa không còn đại diện đúng chất lượng không khí tại vị trí user. Vượt 20km → coi như "chưa có dữ liệu khu vực này", không trả về AQI của trạm xa nhất tìm được |
| **Xử lý thiếu chỉ số phụ (PM10, O3, NO2, SO2, CO)** | **Vẫn lưu record** nếu có `aqi` hợp lệ | Các cột này vốn `NULL`-able trong `aqi_readings` — thiếu 1 vài chỉ số phụ không ảnh hưởng khả năng tính cảnh báo (chỉ dựa vào `aqi` tổng hợp) |
| **Xử lý thiếu chính `aqi`** (WAQI trả `"-"` hoặc giá trị không hợp lệ) | **Bỏ qua, không lưu record** cho chu kỳ đó | Log lại lỗi ở mức WARNING, không làm fail toàn bộ job — các trạm khác vẫn tiếp tục xử lý bình thường |
| **Dữ liệu trạm quá cũ (stale)** | `last_synced_at` **quá 3 giờ** (gấp 3 lần chu kỳ fetch bình thường) → coi là stale | Alert Evaluation **bỏ qua** đánh giá cảnh báo cho user gần trạm đó trong chu kỳ này (không dùng số liệu cũ để cảnh báo sai) |
| **Cảnh báo độ mới dữ liệu trên Dashboard** | Reading cũ hơn **2 giờ** → Frontend hiển thị nhãn "Dữ liệu có thể chưa cập nhật" | Ngưỡng UI cảnh báo (2h) thấp hơn ngưỡng stale dùng cho Alert Evaluation (3h) — cho user biết sớm hơn trước khi hệ thống chính thức ngừng cảnh báo dựa trên dữ liệu đó |

> **Hệ quả gộp quy tắc**: giới hạn 20km ở trên đã tự nhiên xử lý luôn trường hợp "vị trí ngoài phạm vi Hà Nội/TP.HCM" (điểm E.12 đã nêu trước đó) — không cần thêm rule kiểm tra ranh giới thành phố riêng, vì bất kỳ vị trí nào cách xa mọi trạm > 20km đều tự động bị coi là "chưa hỗ trợ khu vực này".

## 4. Gửi cảnh báo đa kênh

| Rule | Giá trị đã chốt | Lý do |
| --- | --- | --- |
| **Ưu tiên kênh** | **Gửi cả 2 kênh** (Web Push + Email) nếu user đã đăng ký cả 2, không phân cấp ưu tiên | Đơn giản cho MVP — mỗi kênh gửi thành công/thất bại độc lập, ghi 1 row riêng trong `alerts_log` (phân biệt bằng cột `channel`) |
| **Xử lý Web Push thất bại (410 Gone / 404)** | Đánh dấu `push_subscriptions.is_active = false` + `last_failed_at = now()` | Không xóa cứng record — giữ lịch sử để debug tỷ lệ thất bại (đã bổ sung ở migration `000011`) |
| **Subscription không active** | Bỏ qua khi gửi, không tính là "thất bại" trong `alerts_log` | Tránh nhiễu log — chỉ ghi `alerts_log` cho các lần **thực sự cố gắng gửi** |

## 5. Giới hạn phạm vi MVP (có chủ đích, không phải thiếu sót)

- **1 user chỉ khai được 1 `condition_type`** (không hỗ trợ đa bệnh lý đồng thời) — nếu user vừa hen suyễn vừa dị ứng, chọn bệnh lý có `base` ngưỡng thấp hơn (`asthma`/`allergic_rhinitis` đều = 100, ngang nhau) để đảm bảo an toàn hơn `none`.
- Không kiểm tra ranh giới hành chính Hà Nội/TP.HCM tường minh — dùng chung rule "khoảng cách trạm ≤ 20km" ở mục 3 để đơn giản hóa, tránh trùng lặp logic.

## 6. Bảng tham chiếu: Rule nào thuộc module nào

| Rule | Module áp dụng | File liên quan (dự kiến) |
| --- | --- | --- |
| Ma trận threshold (mục 1) | `health_profile` | `internal/service/health_profile_service.go` |
| Validation custom threshold (mục 1) | `health_profile` | `internal/handler/health_profile_handler.go` (validate trước khi gọi service) |
| Cooldown & dedup (mục 2) | `alert` | `internal/service/alert_service.go` |
| Khoảng cách trạm hợp lệ (mục 3) | `location`, `aqi` | `internal/repository/station_repo.go` (query `FindNearest` trả kèm khoảng cách để service kiểm tra) |
| Xử lý dữ liệu thiếu/stale (mục 3) | `aqi` | `internal/worker/fetch_aqi_task.go` |
| Đa kênh & subscription thất bại (mục 4) | `notification` | `internal/service/notification_service.go` |

---

**Sau khi chốt file này**: Bước 3 (Domain Modeling & Business Rules) hoàn tất. Có thể tiến sang Bước 4 (Hi-fi UI, có thể làm tắt vì đã có shadcn/ui) hoặc Bước 6 (API & Integration Contract) trước khi quay lại code Ingestion Worker.
