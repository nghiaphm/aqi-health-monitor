import { AQILevel } from "@/types/aqi";
import { getAqiMeta } from "@/lib/constants";
import { AqiBadge } from "./aqi-badge";

interface AqiGaugeProps {
  aqi: number;
  level: AQILevel;
  maxScale?: number;
}

export function AqiGauge({ aqi, level, maxScale = 500 }: AqiGaugeProps) {
  const meta = getAqiMeta(aqi);

  // SVG Gauge calculations (semi-circle or 240 degree arc)
  const radius = 90;
  const strokeWidth = 14;
  const normalizedRadius = radius - strokeWidth / 2;
  const circumference = normalizedRadius * 2 * Math.PI;

  // Arc of 240 degrees (2/3 of a full circle)
  const arcLength = circumference * (240 / 360);
  const strokeDashoffset =
    arcLength - (Math.min(Math.max(aqi, 0), maxScale) / maxScale) * arcLength;

  return (
    <div className="flex flex-col items-center justify-center p-6 relative">
      <div className="relative w-64 h-52 flex items-center justify-center">
        <svg
          className="w-full h-full transform -rotate-[210deg]"
          viewBox="0 0 200 200"
        >
          {/* Background Arc */}
          <circle
            cx="100"
            cy="100"
            r={normalizedRadius}
            fill="transparent"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            strokeDasharray={`${arcLength} ${circumference}`}
            className="text-muted/30"
            strokeLinecap="round"
          />

          {/* Active Value Arc */}
          <circle
            cx="100"
            cy="100"
            r={normalizedRadius}
            fill="transparent"
            stroke={meta.colorHex}
            strokeWidth={strokeWidth}
            strokeDasharray={`${arcLength} ${circumference}`}
            style={{
              strokeDashoffset,
              transition: "stroke-dashoffset 0.8s ease-out, stroke 0.4s ease",
            }}
            strokeLinecap="round"
          />
        </svg>

        {/* Center AQI Number & Level */}
        <div className="absolute inset-0 flex flex-col items-center justify-center pt-4">
          <span className="text-xs uppercase tracking-wider text-muted-foreground font-semibold">
            Chỉ số AQI
          </span>
          <span
            className="text-6xl font-black tracking-tight"
            style={{ color: meta.colorHex }}
          >
            {aqi}
          </span>
          <div className="mt-1">
            <AqiBadge aqi={aqi} level={level} showScore={false} />
          </div>
        </div>
      </div>

      <div className="flex justify-between w-56 text-[11px] text-muted-foreground font-medium -mt-2">
        <span>0 (Tốt)</span>
        <span>500 (Nguy hại)</span>
      </div>
    </div>
  );
}
