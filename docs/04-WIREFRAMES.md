# Wireframes (Low-fi) — AQI Health Monitor

> Tài liệu này tổng hợp **7 wireframe Low-fi** cho các màn hình chính — hoàn thiện phần còn thiếu ở Bước 2 (User Flow & Wireframe) trong quy trình phát triển.
>
> Đây là bản phác thảo **bố cục thô** (không màu sắc, không chi tiết thẩm mỹ) — chỉ nhằm xác nhận cấu trúc thông tin và vị trí các thành phần trên màn hình trước khi bước sang Hi-fi UI/UX (Bước 4), nơi mới quyết định màu sắc, typography, spacing cụ thể theo shadcn/ui.
>
> Mỗi wireframe tương ứng 1-1 với 1 trong 7 User Flow đã định nghĩa ở `USER-FLOWS.md` — xem file đó để biết nhánh rẽ/logic đầy đủ đằng sau mỗi màn hình.

## Mục lục

1. [Login / Landing](#1-login--landing)
2. [Onboarding — Bước 1: Hồ sơ sức khỏe](#2-onboarding--bước-1-hồ-sơ-sức-khỏe)
3. [Onboarding — Bước 2: Vị trí](#3-onboarding--bước-2-vị-trí)
4. [Dashboard chính](#4-dashboard-chính)
5. [History](#5-history)
6. [Settings — Ngưỡng cảnh báo](#6-settings--ngưỡng-cảnh-báo)
7. [Alerts (lịch sử cảnh báo)](#7-alerts-lịch-sử-cảnh-báo)

---

## 1. Login / Landing

**Tương ứng User Flow**: [1 — Đăng ký / Đăng nhập](./USER-FLOWS.md#1-đăng-ký--đăng-nhập)

**Thành phần chính**: logo/tên app ở giữa trên, tagline ngắn giới thiệu sản phẩm, nút "Đăng nhập" cố định ở dưới cùng.

![Wireframe 01 - Login Landing](../assets/wireframe-01.png)

---

## 2. Onboarding — Bước 1: Hồ sơ sức khỏe

**Tương ứng User Flow**: [2 — Onboarding](./USER-FLOWS.md#2-onboarding-thiết-lập-hồ-sơ)

**Thành phần chính**: dropdown chọn bệnh lý (`condition_type`), dropdown chọn nhóm tuổi (`age_group`), nút "Tiếp tục" cố định dưới cùng.

![Wireframe 02 - Onboarding Health Profile](../assets/wireframe-02.png)

---

## 3. Onboarding — Bước 2: Vị trí

**Tương ứng User Flow**: [2 — Onboarding](./USER-FLOWS.md#2-onboarding-thiết-lập-hồ-sơ)

**Thành phần chính**: khung bản đồ placeholder, nút chính "Dùng vị trí hiện tại" (GPS), link phụ "Nhập địa chỉ thủ công" cho nhánh rẽ khi user từ chối cấp quyền GPS.

![Wireframe 03 - Onboarding Location](../assets/wireframe-03.png)

---

## 4. Dashboard chính

**Tương ứng User Flow**: [5 — Xem Dashboard](./USER-FLOWS.md#5-xem-dashboard-aqi-hiện-tại)

**Thành phần chính**: thanh điều hướng trên cùng, gauge hiển thị chỉ số AQI hiện tại, badge mức độ cảnh báo, khung bản đồ vị trí thu nhỏ, khối khuyến nghị hành động ở cuối.

![Wireframe 04 - Dashboard](../assets/wireframe-04.png)

---

## 5. History

**Tương ứng User Flow**: [6 — Xem History](./USER-FLOWS.md#6-xem-history)

**Thành phần chính**: bộ lọc khoảng thời gian (7 ngày / 30 ngày), khung biểu đồ đường thể hiện xu hướng AQI theo thời gian.

![Wireframe 05 - History](../assets/wireframe-05.png)

---

## 6. Settings — Ngưỡng cảnh báo

**Tương ứng User Flow**: [3 — Tùy chỉnh ngưỡng cảnh báo](./USER-FLOWS.md#3-tùy-chỉnh-ngưỡng-cảnh-báo), [4 — Đăng ký Web Push](./USER-FLOWS.md#4-đăng-ký-nhận-web-push)

**Thành phần chính**: slider chỉnh `threshold_aqi`, toggle bật/tắt thông báo đẩy (Web Push), nút "Lưu".

![Wireframe 06 - Settings Threshold](../assets/wireframe-06.png)

---

## 7. Alerts (lịch sử cảnh báo)

**Tương ứng User Flow**: [7 — Xem Alerts](./USER-FLOWS.md#7-xem-alerts-lịch-sử-cảnh-báo)

**Thành phần chính**: danh sách dạng dòng (row), mỗi dòng gồm icon trạng thái, 2 dòng nội dung (mức AQI + trạm kích hoạt), và mốc thời gian tương đối.

![Wireframe 07 - Alerts History](../assets/wireframe-07.png)

---

## Ghi chú

- Các wireframe này **cố tình đơn sắc, không có màu sắc/icon thật** — mục đích duy nhất là xác nhận bố cục và luồng thông tin trước khi đầu tư thời gian vào Hi-fi UI (Bước 4).
- Khi chuyển sang Hi-fi, giữ nguyên vị trí và nhóm thành phần đã xác nhận ở đây; chỉ bổ sung màu sắc, spacing, typography theo design token của shadcn/ui — tránh đổi bố cục giữa chừng gây phát sinh việc thiết kế lại từ đầu.
- Nếu 1 màn hình phát sinh thêm thành phần mới trong lúc code Hi-fi (VD: thêm nút refresh trên Dashboard), cập nhật lại wireframe tương ứng ở đây để tài liệu không lệch pha với sản phẩm thật.
