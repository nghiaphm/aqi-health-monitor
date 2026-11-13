import { AQILevel } from "@/types/aqi";
import { getAqiMeta } from "@/lib/constants";
import { cn } from "@/lib/utils";

interface AqiBadgeProps {
  aqi?: number;
  level?: AQILevel;
  className?: string;
  showScore?: boolean;
}

export function AqiBadge({
  aqi,
  level,
  className,
  showScore = true,
}: AqiBadgeProps) {
  const meta = getAqiMeta(aqi !== undefined ? aqi : level || "good");

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold shadow-xs border",
        meta.badgeClass,
        className
      )}
    >
      <span className="w-2 h-2 rounded-full bg-current opacity-80" />
      {showScore && aqi !== undefined && <span>{aqi} ·</span>}
      <span>{meta.label}</span>
    </span>
  );
}
