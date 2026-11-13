import { CurrentAQIResponse } from "@/types/aqi";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { MapPin, Clock, ShieldAlert, Wind } from "lucide-react";

interface AqiCardProps {
  data: CurrentAQIResponse;
}

export function AqiCard({ data }: AqiCardProps) {
  const { station, reading, advice } = data;

  // Format measured_at time
  let formattedTime = reading.measured_at;
  try {
    const d = new Date(reading.measured_at);
    formattedTime = new Intl.DateTimeFormat("vi-VN", {
      hour: "2-digit",
      minute: "2-digit",
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      timeZone: "Asia/Ho_Chi_Minh",
    }).format(d);
  } catch {
    // Keep raw
  }

  const pollutantItems = [
    { key: "PM2.5", value: reading.pollutants.pm25, unit: "µg/m³", isDominant: reading.dominant_pollutant === "pm25" },
    { key: "PM10", value: reading.pollutants.pm10, unit: "µg/m³", isDominant: reading.dominant_pollutant === "pm10" },
    { key: "O₃", value: reading.pollutants.o3, unit: "ppb", isDominant: reading.dominant_pollutant === "o3" },
    { key: "NO₂", value: reading.pollutants.no2, unit: "ppb", isDominant: reading.dominant_pollutant === "no2" },
    { key: "SO₂", value: reading.pollutants.so2, unit: "ppb", isDominant: reading.dominant_pollutant === "so2" },
    { key: "CO", value: reading.pollutants.co, unit: "ppm", isDominant: reading.dominant_pollutant === "co" },
  ];

  return (
    <div className="space-y-6">
      {/* Station Metadata */}
      <div className="flex flex-wrap items-center justify-between text-xs text-muted-foreground gap-3 px-1">
        <div className="flex items-center gap-1.5">
          <MapPin className="w-4 h-4 text-primary shrink-0" />
          <span>
            Trạm đo: <strong className="text-foreground">{station.name}</strong>
            {station.city ? ` (${station.city})` : ""} · cách {station.distance_m >= 1000 ? `${(station.distance_m / 1000).toFixed(1)} km` : `${station.distance_m} m`}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <Clock className="w-4 h-4 shrink-0" />
          <span>Đo lúc: {formattedTime}</span>
        </div>
      </div>

      {/* 6 Pollutants Grid */}
      <Card className="border shadow-xs">
        <CardHeader className="py-3 px-4 border-b">
          <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-semibold flex items-center gap-1.5">
            <Wind className="w-4 h-4 text-primary" />
            Chi tiết các chất ô nhiễm
          </CardTitle>
        </CardHeader>
        <CardContent className="p-4 grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3 text-center">
          {pollutantItems.map((p) => (
            <div
              key={p.key}
              className={`p-2.5 rounded-xl border flex flex-col items-center justify-center ${
                p.isDominant ? "bg-primary/5 border-primary/30 ring-1 ring-primary/20" : "bg-muted/10 border-border"
              }`}
            >
              <div className="flex items-center gap-1">
                <span className="text-xs font-semibold text-foreground">{p.key}</span>
                {p.isDominant && (
                  <span className="text-[10px] bg-primary/20 text-primary px-1 rounded-sm font-medium">
                    Chính
                  </span>
                )}
              </div>
              <span className="text-lg font-bold text-foreground mt-0.5">
                {p.value !== null && p.value !== undefined ? p.value : "—"}
              </span>
              <span className="text-[10px] text-muted-foreground">{p.unit}</span>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Advice Section */}
      <Card className="border shadow-xs bg-card">
        <CardHeader className="py-3 px-4 border-b">
          <CardTitle className="text-sm font-semibold flex items-center gap-2">
            <ShieldAlert className="w-4 h-4 text-primary" />
            Khuyến nghị sức khỏe dành cho bạn
          </CardTitle>
        </CardHeader>
        <CardContent className="p-4 space-y-3">
          <p className="text-sm font-medium leading-relaxed text-foreground">
            {advice.summary}
          </p>

          <div className="space-y-2 pt-2 border-t border-border">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Hành động nên làm:
            </span>
            <ul className="space-y-2">
              {advice.actions.map((act, idx) => (
                <li key={idx} className="flex items-start gap-2 text-xs text-foreground leading-normal">
                  <span className="w-1.5 h-1.5 rounded-full bg-primary mt-1.5 shrink-0" />
                  <span>{act}</span>
                </li>
              ))}
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
