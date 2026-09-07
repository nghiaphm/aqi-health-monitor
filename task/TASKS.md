# Tasks — AQI Health Monitor

> **File này là "cửa sổ chính"** — luôn giữ nhỏ gọn, không copy nội dung chi tiết từ nơi khác vào đây. Đọc file này để biết **đang ở đâu**; đọc file khác khi cần **chi tiết**.
>
> - Task đang làm (checklist cụ thể) → xem `task/CURRENT-TASK.md` (bị ghi đè mỗi khi chuyển bước, không cộng dồn)
> - Lịch sử quyết định/sự cố → xem `task/CHANGELOG.md` (append-only, chỉ đọc khi cần tra cứu quá khứ)
> - Chi tiết từng bước đã xong → không cần file archive riêng, vì mỗi bước đã có sẵn tài liệu chính thức của nó (`RESEARCH.md`, `ARCHITECTURE.md`...) — xem cột "Tài liệu" ở bảng dưới.

## 0. Quy tắc làm việc đã rút ra (đọc trước khi code)

1. **Tài liệu dẫn dắt code, không phải ngược lại.** Sửa tài liệu trước, code sau.
2. **Không tự quyết định số liệu nghiệp vụ ngay trong lúc code.** Tra `BUSINESS-RULES.md` trước; chưa có thì bổ sung vào đó trước.
3. **Trước khi coi 1 bước "xong", đối chiếu chéo với các tài liệu khác.** Từng thiếu `alerts_log.location_id` vì không rà lại `BUSINESS-RULES.md` trước khi chốt DB.
4. **Đừng tạo thêm tài liệu chỉ để "cho đủ lý thuyết SDLC".** Luôn hỏi: file này thêm thông tin mới, hay chỉ diễn đạt lại?
5. **User Flow ≠ System Flow.** Gắn nhãn 🔧 Background / 👤 User-driven rõ ràng nếu 1 tài liệu trộn cả 2 loại.
6. **Wireframe Low-fi phải đơn sắc, không màu** — giữ ranh giới rõ với Hi-fi.
7. **Tận dụng Design System có sẵn (shadcn/ui) = đã xong phần lớn Hi-fi** — chỉ cần bổ sung token đặc thù domain.
8. **Lỗi Docker "exec format error" không nhất thiết là sai kiến trúc CPU** — luôn xem log trước khi kết luận nguyên nhân.
9. **Mọi migration mới phải cập nhật bảng "Lịch sử thay đổi schema"** trong `DATABASE_DESIGN.md`.
10. **File task phải tách "index nhẹ" khỏi "nhật ký nặng dần"** — không để 1 file phình to vô hạn rồi tốn ngữ cảnh mỗi lần đọc lại.

## 1. Trạng thái tổng quan theo 12 bước

| # | Bước | Trạng thái | Tài liệu |
| --- | --- | --- | --- |
| 1 | Requirement & Business Analysis | ✅ Xong | `RESEARCH.md` |
| 2 | User Flow & Wireframe (Low-fi) | ✅ Xong | `SYSTEM-FLOW.md`, `USER-FLOWS.md`, `WIREFRAMES.md` |
| 3 | Domain Modeling & Business Rules | ✅ Xong | `BUSINESS-RULES.md`, `DATABASE_DESIGN.md` |
| 4 | UI/UX Design (Hi-fi) | ✅ Xong (rút gọn có chủ đích) | `DESIGN-DECISIONS.md` |
| 5 | System & Database Architecture | ✅ Xong | `ARCHITECTURE.md`, `DATABASE_DESIGN.md`, `migrations/000001–000011` |
| 6 | API & Integration Contract Design | ✅ Xong | `API-CONTRACT.md` = `docs/09-API_CONTRACT.md` |
| 7 | Vertical Slice (Tracer Bullet) | ⬜ Chưa bắt đầu | — |
| 8 | Continuous Implementation | ⬜ Chưa bắt đầu | — |
| 9 | Integrated Security & Testing | ⬜ Chưa bắt đầu | — |
| 10 | CI/CD & Deployment | 🔄 Một phần | `docker-compose.yml`, `Dockerfile` đã có; GitHub Actions chưa viết |
| 11 | Observability & Monitoring | ⬜ Chưa bắt đầu | — |
| 12 | Documentation, Release & Handoff | 🔄 Liên tục | Toàn bộ tài liệu — cập nhật song song |

## 2. Trạng thái hạ tầng hiện tại

| Hạng mục | Trạng thái | Ghi chú |
| --- | --- | --- |
| Repo GitHub | ✅ Đã tạo, đã push scaffold | `chore/initial-project-scaffold` |
| `docker-compose.yml` | ✅ Đã viết | postgres, redis, keycloak+db, backend-api, backend-worker, frontend |
| `Dockerfile` backend/frontend | ✅ Đã viết | Backend multi-stage (`api`/`worker`); frontend cần `output: "standalone"` |
| Migration DB | ✅ `000001` → `000011` đã viết | **Cần xác nhận đã `migrate up` đủ 11 file chưa** |
| Keycloak Realm/Client | ⚠️ Chưa xác nhận | Cần confirm trước khi code `auth_middleware.go` |
| GitHub Actions CI/CD | ❌ Chưa viết | Thuộc Bước 10, chưa tới lúc |

## 3. Rủi ro / câu hỏi mở

> Xóa dòng ngay khi đã giải quyết — không để tồn đọng.

| Vấn đề | Cần làm gì |
| --- | --- |
| Chưa xác nhận Keycloak Realm/Client đã tạo | Kiểm tra Admin Console (`localhost:8080`) trước khi code auth |
| Bán kính 20km, cooldown 6h là giả định chưa kiểm chứng | Sau khi `sync_stations` chạy lần đầu, đối chiếu mật độ trạm thực tế |
| Ma trận threshold là suy luận đơn giản hóa, không phải số liệu y tế chính thức | Ghi rõ trong README/portfolio, tránh trình bày như khuyến nghị y khoa |
