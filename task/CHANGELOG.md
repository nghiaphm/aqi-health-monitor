# Changelog

> Append-only — chỉ thêm dòng mới, không sửa/xóa dòng cũ. Ghi khi có **quyết định quan trọng, đổi hướng, hoặc sự cố đáng nhớ** — không ghi việc nhỏ nhặt hàng ngày. File này không nằm trong đường đọc mặc định của agent; chỉ mở khi cần tra lại lịch sử.

| Ngày | Thay đổi |
|---|---|
| _(điền ngày)_ | Tách `TASKS.md` (1 file, phình to dần) thành 3 file: `TASKS.md` (index nhẹ, ổn định), `CURRENT-TASK.md` (checklist bước đang làm, bị ghi đè khi chuyển bước), `CHANGELOG.md` (nhật ký append-only, không đọc mặc định) — lý do: tránh tốn ngữ cảnh khi agent/người đọc lại file theo thời gian dự án kéo dài |
| 2026-09-07 | Hoàn thành Bước 6 — API Contract cho Vertical Slice 4 endpoint (`GET /me`, `POST /health-profile`, `POST /locations`, `GET /aqi/current`) → `docs/09-API_CONTRACT.md`: quy ước chung (versioning `/api/v1`, UUID string, ISO 8601 UTC, format success/lỗi), bảng mã lỗi nghiệp vụ đối chiếu `BUSINESS-RULES.md`, DTO + JSON mẫu từng endpoint, chuỗi file tương tác theo tầng handler→service→repository, đối chiếu đủ nhánh rẽ 7 flow trong `USER-FLOWS.md` |
