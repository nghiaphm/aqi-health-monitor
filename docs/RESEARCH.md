# Research Module — AQI Health Alert Platform

## 1. Khái niệm cơ bản

### 1.1. AQI (Air Quality Index) là gì?

**AQI (Chỉ số Chất lượng Không khí)** là một thước đo chuẩn hóa được sử dụng để định lượng và báo cáo mức độ sạch hoặc ô nhiễm của không khí hàng ngày, cùng với các ảnh hưởng sức khỏe tương ứng đối với cộng đồng.

#### Các chất gây ô nhiễm chính được đo lường

AQI được tính toán dựa trên 5 chất gây ô nhiễm phổ biến theo tiêu chuẩn của Cơ quan Bảo vệ Môi trường Hoa Kỳ (US EPA) và Tổ chức Y tế Thế giới (WHO):

- **PM2.5 & PM10**: Bụi mịn và bụi siêu mịn có đường kính dưới 2.5 µm và 10 µm. Đây là yếu tố ô nhiễm nguy hại nhất cho sức khỏe con người.
- **O3 (Ozone mặt đất)**: Hình thành do phản ứng hóa học giữa ánh nắng mặt trời và các chất ô nhiễm từ xe cộ, nhà máy.
- **CO (Carbon Monoxide)**: Khí không màu, không mùi phát thải từ động cơ đốt trong.
- **SO2 (Sulfur Dioxide)**: Phát thải chính từ các nhà máy điện và khu công nghiệp đốt than/dầu.
- **NO2 (Nitrogen Dioxide)**: Phát thải từ khí thải giao thông và sản xuất công nghiệp.

#### Nguyên lý tính toán và thang đo

- **Cách tính**: mỗi chất ô nhiễm được đo nồng độ theo thời gian thực (hoặc trung bình nhiều giờ), quy đổi sang chỉ số AQI riêng. **AQI tổng hợp của một khu vực = giá trị AQI cao nhất trong số các chất ô nhiễm đó.**
- **Thang đo**: chạy từ **0 đến 500**. Giá trị càng cao, ô nhiễm càng nặng và rủi ro sức khỏe càng lớn.

| Khoảng AQI | Mức cảnh báo | Ý nghĩa & ảnh hưởng sức khỏe |
| --- | --- | --- |
| 0–50 | Tốt | Đạt chuẩn, không ảnh hưởng sức khỏe |
| 51–100 | Trung bình | Chấp nhận được; nhóm rất nhạy cảm có thể bị ảnh hưởng nhẹ |
| 101–150 | Kém | Nhóm nhạy cảm (trẻ em, người già, bệnh hô hấp) bắt đầu có triệu chứng |
| 151–200 | Xấu | Toàn bộ cộng đồng bắt đầu bị ảnh hưởng; nhóm nhạy cảm nghiêm trọng hơn |
| 201–300 | Rất xấu | Cảnh báo khẩn cấp; mọi người đều có nguy cơ |
| 301–500 | Nguy hiểm | Mức cảnh báo cao nhất; toàn bộ dân số ảnh hưởng nghiêm trọng |

### 1.2. Thực trạng tại các đô thị lớn Việt Nam

Hà Nội và TP.HCM thường xuyên ghi nhận AQI ở mức **Kém (101–150)** đến **Xấu (151–200)**. Ô nhiễm nặng tập trung nhiều vào mùa đông ở miền Bắc do hiện tượng **nghịch nhiệt (temperature inversion)** — lớp không khí lạnh bị mắc kẹt dưới lớp không khí ấm, ngăn cản sự phát tán của bụi mịn PM2.5.

#### Bất cập trong cách tiếp cận dữ liệu AQI hiện tại

- **Thiếu cá nhân hóa**: hệ thống cảnh báo hiện tại chỉ đưa ra khuyến cáo chung, trong khi ngưỡng kích hoạt triệu chứng ở người hen suyễn mãn tính khác hoàn toàn với người dị ứng thời tiết hay vận động viên ngoài trời.
- **Thiếu chủ động**: người dùng phải tự mở app kiểm tra, dễ bỏ lỡ thời điểm chất lượng không khí biến động đột ngột.
- **Thiếu ngữ cảnh hành động**: cảnh báo dừng ở con số ("AQI 165") mà không gắn với khuyến nghị hành động cụ thể theo lịch trình cá nhân.

#### Vấn đề cốt lõi (Problem Statement)

> Dữ liệu AQI công khai mang tính tổng quan và thụ động, trong khi nhu cầu bảo vệ sức khỏe mang tính cá nhân và thời gian thực. Thiếu một công cụ **cá nhân hóa ngưỡng cảnh báo** theo hồ sơ sức khỏe riêng và **chủ động thông báo đúng thời điểm**.

### 1.3. Mục tiêu giải pháp

Chuyển đổi từ tra cứu thụ động sang **cảnh báo chủ động, cá nhân hóa** dựa trên vị trí địa lý và hồ sơ sức khỏe riêng của từng người.

**Các thành phần cốt lõi:**

- **Hồ sơ & ngưỡng an toàn cá nhân**: thiết lập ngưỡng cảnh báo AQI riêng theo tình trạng sức khỏe (hen suyễn, dị ứng, người già/trẻ nhỏ) thay vì dùng thang đo chung.
- **Đồng bộ dữ liệu & vị trí**: tự động lấy AQI từ trạm đo công khai gần nhất dựa trên tọa độ người dùng.
- **Cảnh báo chủ động**: gửi thông báo (web push / email) ngay khi AQI tại vị trí hiện tại vượt ngưỡng an toàn đã cài đặt.
- **Khuyến nghị hành động**: chỉ dẫn cụ thể (đeo khẩu trang N95, bật máy lọc không khí, hủy lịch tập ngoài trời).

**Luồng xử lý ngắn gọn:**

```
Tọa độ GPS + Dữ liệu AQI công khai → so sánh với Ngưỡng sức khỏe cá nhân → Gửi cảnh báo & Khuyến nghị hành động
```

**Giá trị cốt lõi**: giúp người dùng phòng ngừa rủi ro sức khỏe kịp thời mà không cần tự tra cứu hay đánh giá thủ công.

> **Lưu ý phạm vi (điều chỉnh so với bản nháp gốc)**: kênh cảnh báo MVP là **Web Push Notification + Email**, khớp với stack đã chọn (Next.js frontend, Resend cho email). Build ứng dụng mobile riêng (React Native/Flutter) là hướng mở rộng có thể cân nhắc sau, không thuộc phạm vi MVP hiện tại.

---

## 2. Các mục nghiên cứu chính

### 2.1. Nguồn dữ liệu (Data Source)

**Khả năng đáp ứng tại Việt Nam**:

- WAQI tổng hợp dữ liệu thời gian thực từ trạm quan trắc quốc gia, đại sứ quán/lãnh sự quán Hoa Kỳ, và các trạm cộng đồng (PurpleAir, PAM Air).
- Khu vực hỗ trợ: phủ rộng tại Hà Nội, TP.HCM, Đà Nẵng và nhiều tỉnh/thành lớn.
- Thông số trả về: AQI tổng hợp, chi tiết từng chỉ số thành phần (PM2.5, PM10, O3, NO2, SO2, CO), nhiệt độ, độ ẩm, thông tin trạm đo.

**Đăng ký API Key & giới hạn Rate Limit**:

- API Key miễn phí: đăng ký qua form tại `aqicn.org/data-platform/token/`.
- Hạn ngạch mặc định: **1.000 requests/giây** — hoàn toàn dư sức cho nhu cầu MVP.
- Ràng buộc điều khoản quan trọng:
  - Miễn phí cho mục đích phi thương mại/nghiên cứu; **không được bán hoặc đưa dữ liệu vào gói trả phí**.
  - **Không được lưu trữ/redistribute dữ liệu dưới dạng cache hoặc archive** để phân phối lại cho bên thứ ba — cache trong kiến trúc của dự án chỉ phục vụ mục đích vận hành nội bộ (giảm số lần gọi API), không dùng để "bán lại" dữ liệu.
  - Bắt buộc ghi nhận nguồn (attribution) tới World Air Quality Index Project và EPA gốc.

**Các endpoint chính cần test cho dự án**:

| Mục đích | Endpoint REST | Tham số |
| --- | --- | --- |
| AQI theo tọa độ GPS | `/feed/geo::lat;:lng/?token=:token` | lat, lng của người dùng |
| AQI theo tên thành phố | `/feed/:city/?token=:token` | VD: `hanoi`, `ho-chi-minh-city` |
| Tìm trạm đo trong khu vực | `/map/bounds/?latlng=:latlng&token=:token` | Bounding box tọa độ (lat1,lng1,lat2,lng2) |

**Đánh giá & lưu ý kỹ thuật cho Backend**:

- **Ưu điểm**: endpoint đơn giản, phản hồi JSON nhẹ, hỗ trợ tìm trạm gần nhất theo tọa độ chính xác cho tính năng cá nhân hóa vị trí.
- **Nhược điểm**: tốc độ cập nhật dữ liệu gốc phụ thuộc từng trạm (thường 1 giờ/lần).
- **Giải pháp kiến trúc**: dùng Redis cache cho các tọa độ/khu vực trùng lặp để giảm số request không cần thiết và tăng tốc phản hồi — cache có TTL ngắn (phục vụ vận hành), không lưu trữ vĩnh viễn để tuân thủ điều khoản sử dụng.

### 2.2. Chuẩn y tế (Domain Knowledge)

Dự án tham chiếu khung phân ngưỡng của **US EPA** kết hợp với hướng dẫn chất lượng không khí của **WHO**.

**Phân khoảng AQI và ngưỡng tác động sức khỏe**:

| Khoảng AQI | Mức cảnh báo | Nhóm nhạy cảm | Tác động y tế & khuyến nghị |
| --- | --- | --- | --- |
| 0–50 | Tốt | Không có | An toàn cho mọi hoạt động ngoài trời |
| 51–100 | Trung bình | Người cực nhạy cảm | Bắt đầu ảnh hưởng nhẹ đến người siêu nhạy cảm |
| 101–150 | Kém | Trẻ em, người già, bệnh hô hấp/tim mạch | **Kích hoạt cảnh báo**: nhóm nhạy cảm có triệu chứng (khó thở, ho); hạn chế vận động mạnh ngoài trời |
| 151–200 | Xấu | Tất cả mọi người | **Kích hoạt cảnh báo toàn bộ**: nhóm nhạy cảm bị ảnh hưởng nghiêm trọng; tránh ra ngoài, đeo khẩu trang N95 |
| 201–300 | Rất xấu | Tất cả mọi người | Cảnh báo khẩn cấp; hạn chế tối đa ra ngoài |
| 301–500 | Nguy hiểm | Tất cả mọi người | Cảnh báo cao nhất; ở trong nhà, bật máy lọc không khí |

**Cơ sở khoa học cho logic phân nhóm (User Profiling)**:

- **PM2.5** là chất ô nhiễm cốt lõi gây kích ứng đường hô hấp — hạt bụi kích thước dưới 2.5 µm đi sâu vào phế nang và vào máu.
- **Ngưỡng kích hoạt (trigger threshold) theo nhóm:**
  - Người bình thường: mặc định AQI > 150.
  - Trẻ em, người cao tuổi: hệ miễn dịch/phổi yếu hơn → ngưỡng AQI > 100.
  - Bệnh nhân hô hấp/tim mạch/thai phụ: độ nhạy cao → ngưỡng AQI > 100 (có thể tùy chỉnh xuống AQI > 80 nếu tiền sử nặng).

### 2.3. Phạm vi sản phẩm (MVP Scope)

**Phạm vi địa lý**: **Hà Nội** và **TP. Hồ Chí Minh** — hai đô thị mật độ dân số cao nhất, tần suất AQI mức Kém–Xấu cao, và có mật độ trạm quan trắc (công cộng & cộng đồng) dày đặc nhất.

**Phạm vi bệnh lý & nhóm người dùng**: tập trung 2 nhóm bệnh lý hô hấp phổ biến nhất chịu ảnh hưởng trực tiếp bởi PM2.5:

1. Hen suyễn (Asthma)
2. Viêm mũi dị ứng / dị ứng thời tiết (Allergic Rhinitis)

Bổ sung thiết lập nhanh cho 2 đối tượng ưu tiên: **trẻ em** và **người cao tuổi**.

**Kênh phát cảnh báo**: **Web Push Notification + Email** (Resend) — phù hợp với frontend Next.js đã chọn, độ trễ thấp, không phát sinh chi phí SMS Gateway, không cần build app mobile riêng ở giai đoạn MVP.

**Bảng tóm tắt phạm vi MVP**:

| Hạng mục | Trong phạm vi (MVP) | Ngoài phạm vi (phát triển sau) |
| --- | --- | --- |
| Khu vực | Hà Nội, TP.HCM | Các tỉnh/thành khác |
| Bệnh lý | Hen suyễn, viêm mũi dị ứng | Tim mạch, thai phụ, COPD |
| Kênh cảnh báo | Web Push + Email | Mobile app, SMS, Zalo Notification Service |
| Cơ chế vị trí | GPS hiện tại + 1 địa điểm cố định (nhà) | Tự động quét theo hành trình di chuyển liên tục |

### 2.4. Công nghệ cốt lõi (Technical Feasibility)

Lựa chọn **PostGIS** và **TimescaleDB** (extension trên PostgreSQL) giải quyết đúng hai bài toán kỹ thuật trọng tâm — **truy vấn không gian** và **lưu trữ time-series** — mà không phình to hạ tầng không cần thiết.

| Công nghệ | Bài toán cốt lõi | Lý do chọn (đúng & đủ) | Tránh lãng phí/thừa thãi |
| --- | --- | --- | --- |
| **PostGIS** (Geospatial extension) | Xác định trạm đo gần nhất theo tọa độ GPS người dùng | Hàm `ST_DWithin`/`ST_Distance` tính khoảng cách trên mặt cầu nhanh, chính xác; tích hợp trực tiếp vào PostgreSQL, không cần server GIS riêng | Không cần Spatial Search Engine phức tạp (Elasticsearch geospatial) → giảm độ phức tạp hạ tầng, giảm chi phí RAM/server cho MVP |
| **TimescaleDB** (Time-series extension) | Lưu trữ & truy vấn lịch sử AQI biến động liên tục theo giờ/ngày từ nhiều trạm | Hypertables tự động chia nhỏ dữ liệu theo thời gian (time-bucket), giữ tốc độ ghi/đọc ổn định khi dữ liệu phình to; hỗ trợ nén dữ liệu tự động (giảm ~90% dung lượng đĩa); dùng SQL chuẩn để query trend | Không cần NoSQL/time-series DB rời (InfluxDB, Prometheus) → tránh quản lý 2 database độc lập, giữ nguyên ACID transaction của PostgreSQL |
