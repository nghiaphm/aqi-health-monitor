# Design Decisions — AQI Health Monitor

> Tài liệu này chốt các quyết định thiết kế **đặc thù cho domain AQI** mà shadcn/ui không có sẵn mặc định — hoàn thiện phần còn thiếu ở Bước 4 (Hi-fi UI/UX), sau khi đã mượn Design System nền từ shadcn/ui + Radix UI.

## 1. Bảng màu AQI (AQI Color Scale)

Dùng đúng bảng màu chính thức **US-EPA / AirNow** — nhất quán với chính nguồn dữ liệu (WAQI dùng chuẩn EPA), không tự định nghĩa màu riêng.

| AQI | Mức (VN) | Màu gốc EPA | Background (badge/card) | Text/Icon color | Dark mode background |
| --- | --- | --- | --- | --- | --- |
| 0–50 | Tốt | Green | `#00E400` | `#0A3D00` (đen ngả xanh, đủ tương phản) | `#1B5E00` |
| 51–100 | Trung bình | Yellow | `#F5C518` (đã tối hơn `#FFFF00` gốc) | `#3D3000` | `#8A6D00` |
| 101–150 | Kém | Orange | `#FF7E00` | `#3D1F00` | `#B35600` |
| 151–200 | Xấu | Red | `#FF0000` | `#FFFFFF` | `#B30000` |
| 201–300 | Rất xấu | Purple | `#8F3F97` | `#FFFFFF` | `#6B2F72` |
| 300+ | Nguy hiểm | Maroon | `#7E0023` | `#FFFFFF` | `#5C001A` |

**Lý do chỉnh riêng mức "Trung bình"**: vàng thuần `#FFFF00` không đạt tương phản WCAG AA với cả chữ đen lẫn trắng khi dùng làm nền badge. Đã hạ độ sáng xuống `#F5C518`, vẫn giữ đúng "cảm giác vàng" nhận diện AQI nhưng đọc được chữ đen lên trên. Các mức còn lại giữ nguyên hex gốc EPA vì đã đủ tương phản.

### Khai báo vào Tailwind (mở rộng theme cho shadcn/ui)

```js
// tailwind.config.ts
export default {
  theme: {
    extend: {
      colors: {
        aqi: {
          good: { DEFAULT: "#00E400", fg: "#0A3D00" },
          moderate: { DEFAULT: "#F5C518", fg: "#3D3000" },
          unhealthySensitive: { DEFAULT: "#FF7E00", fg: "#3D1F00" },
          unhealthy: { DEFAULT: "#FF0000", fg: "#FFFFFF" },
          veryUnhealthy: { DEFAULT: "#8F3F97", fg: "#FFFFFF" },
          hazardous: { DEFAULT: "#7E0023", fg: "#FFFFFF" },
        },
      },
    },
  },
};
```

### Hàm map AQI → token màu (dùng chung Frontend, đặt tại `lib/constants.ts`)

```ts
// lib/constants.ts
export function getAqiColorToken(aqi: number) {
  if (aqi <= 50) return "aqi-good";
  if (aqi <= 100) return "aqi-moderate";
  if (aqi <= 150) return "aqi-unhealthySensitive";
  if (aqi <= 200) return "aqi-unhealthy";
  if (aqi <= 300) return "aqi-veryUnhealthy";
  return "aqi-hazardous";
}
```

> Ngưỡng số ở đây (50/100/150/200/300) phải luôn khớp với bảng phân loại đã dùng trong `RESEARCH.md` và `BUSINESS-RULES.md` — nếu 1 nơi đổi, 3 nơi phải đổi theo.

## 2. Quy tắc hiển thị số liệu lớn (AQI Gauge)

AQI hiện tại là **con số quan trọng nhất màn hình Dashboard** — không thể trình bày như text thường.

| Thuộc tính | Quy tắc |
| --- | --- |
| Kích thước chữ | Lớn nhất trên toàn màn hình (tối thiểu gấp 4 lần cỡ chữ body) |
| Vị trí | Trung tâm màn hình hoặc ngay dưới thanh điều hướng — không đặt lẫn trong danh sách |
| Màu số | Dùng đúng `text` color theo bảng mục 1 tương ứng mức AQI hiện tại |
| Vòng gauge (nếu dùng dạng vòng tròn) | Progress ring tô theo `background` color tương ứng, phần chưa đầy dùng màu xám trung tính (`muted` của shadcn), không dùng gradient nhiều màu gây rối mắt |
| Nhãn mức độ đi kèm | Luôn hiển thị **chữ** ("Kém", "Xấu"...) cạnh con số — không chỉ dựa vào màu, vì user có thể bị mù màu (đúng nguyên tắc accessibility: không truyền tải thông tin chỉ bằng màu sắc) |
| Thư viện gợi ý | Không có sẵn trong shadcn/ui — dùng SVG tự vẽ (đơn giản, kiểm soát màu chính xác theo bảng trên) hoặc `react-circular-progressbar` nếu muốn animation có sẵn |

## 3. Quy ước Badge mức độ AQI (dùng lại nhiều nơi: Dashboard, History, Alerts)

Tạo 1 component dùng chung `<AqiBadge aqi={value} />` (đặt tại `components/aqi/aqi-badge.tsx`) thay vì lặp lại logic map màu ở từng trang — mọi nơi hiển thị mức AQI (Dashboard, dòng trong Alerts, tooltip trên History chart) đều phải dùng chung component này để đảm bảo nhất quán màu sắc toàn app.

```tsx
// components/aqi/aqi-badge.tsx (định hướng, chưa phải code đầy đủ)
import { cn } from "@/lib/utils";
import { getAqiColorToken } from "@/lib/constants";

// Map trực tiếp ra class đầy đủ để Tailwind JIT scanner nhận diện được
const colorMap: Record<string, string> = {
  "aqi-good": "bg-aqi-good text-aqi-good-fg",
  "aqi-moderate": "bg-aqi-moderate text-aqi-moderate-fg",
  "aqi-unhealthySensitive": "bg-aqi-unhealthySensitive text-aqi-unhealthySensitive-fg",
  "aqi-unhealthy": "bg-aqi-unhealthy text-aqi-unhealthy-fg",
  "aqi-veryUnhealthy": "bg-aqi-veryUnhealthy text-aqi-veryUnhealthy-fg",
  "aqi-hazardous": "bg-aqi-hazardous text-aqi-hazardous-fg",
};

export function AqiBadge({ aqi }: { aqi: number }) {
  const token = getAqiColorToken(aqi);
  return (
    <span className={cn("px-3 py-1 rounded-full text-sm font-medium", colorMap[token])}>
      {aqi} · {getAqiLabel(aqi)}
    </span>
  );
}
```

## 4. Hi-fi Mockup mẫu — Dashboard

Áp dụng bảng màu mục 1 và quy tắc hiển thị số liệu mục 2 vào đúng **1 màn hình đại diện** — dùng làm chuẩn tham chiếu khi code 6 màn còn lại, thay vì vẽ Hi-fi đầy đủ cho cả 7 màn (xem lý do ở mục 5).

![Hi-fi Mockup - Dashboard](../assets/hifi-dashboard.png)

**Đối chiếu với `WIREFRAMES.md` mục 4**: giữ đúng bố cục đã chốt (top bar → gauge → vị trí → map → khuyến nghị), chỉ tô màu theo token. Có phát sinh 1 chi tiết mới so với wireframe gốc — hàng chỉ số phụ (PM2.5/PM10/O3) ở cuối màn hình — cần cập nhật ngược lại `WIREFRAMES.md` mục 4 để 2 tài liệu không lệch nhau.

**Quy tắc áp dụng khi code 6 màn còn lại**: dùng chung style card (`var(--surface-1)`, bo góc `var(--radius)`), cùng kiểu badge bo tròn (`border-radius: 999px`), cùng bộ icon (nếu code bằng `lucide-react`, đối chiếu tên icon tương đương với bộ Tabler dùng trong mockup này).

## 5. Phạm vi KHÔNG làm ở bước này (có chủ đích)

- **Không vẽ Hi-fi mockup đầy đủ cho cả 7 màn hình** — chỉ dựng mẫu cho đúng 1 màn hình đại diện (mục 4), tránh trùng công sức với việc code trực tiếp bằng shadcn/ui.
- **Không tự thiết kế lại spacing/typography scale** — dùng nguyên mặc định của shadcn/ui + Tailwind, vì không có lý do nghiệp vụ nào (khác AQI color) đòi hỏi phải khác biệt.
