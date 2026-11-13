import { AQILevel } from "@/types/aqi";

export interface AQILevelMeta {
  level: AQILevel;
  label: string;
  min: number;
  max: number;
  colorHex: string;
  fgHex: string;
  badgeClass: string;
}

export const AQI_LEVELS: Record<AQILevel, AQILevelMeta> = {
  good: {
    level: "good",
    label: "Tốt",
    min: 0,
    max: 50,
    colorHex: "#00E400",
    fgHex: "#0A3D00",
    badgeClass: "bg-[#00E400] text-[#0A3D00] border-[#00C400]",
  },
  moderate: {
    level: "moderate",
    label: "Trung bình",
    min: 51,
    max: 100,
    colorHex: "#F5C518",
    fgHex: "#3D3000",
    badgeClass: "bg-[#F5C518] text-[#3D3000] border-[#D4A50E]",
  },
  unhealthy_sensitive: {
    level: "unhealthy_sensitive",
    label: "Kém (Nhạy cảm)",
    min: 101,
    max: 150,
    colorHex: "#FF7E00",
    fgHex: "#3D1F00",
    badgeClass: "bg-[#FF7E00] text-[#FFFFFF] border-[#D96B00]",
  },
  unhealthy: {
    level: "unhealthy",
    label: "Xấu",
    min: 151,
    max: 200,
    colorHex: "#FF0000",
    fgHex: "#FFFFFF",
    badgeClass: "bg-[#FF0000] text-[#FFFFFF] border-[#CC0000]",
  },
  very_unhealthy: {
    level: "very_unhealthy",
    label: "Rất xấu",
    min: 201,
    max: 300,
    colorHex: "#8F3F97",
    fgHex: "#FFFFFF",
    badgeClass: "bg-[#8F3F97] text-[#FFFFFF] border-[#723279]",
  },
  hazardous: {
    level: "hazardous",
    label: "Nguy hại",
    min: 301,
    max: 500,
    colorHex: "#7E0023",
    fgHex: "#FFFFFF",
    badgeClass: "bg-[#7E0023] text-[#FFFFFF] border-[#5A0019]",
  },
};

export function getAqiLevelFromScore(aqi: number): AQILevel {
  if (aqi <= 50) return "good";
  if (aqi <= 100) return "moderate";
  if (aqi <= 150) return "unhealthy_sensitive";
  if (aqi <= 200) return "unhealthy";
  if (aqi <= 300) return "very_unhealthy";
  return "hazardous";
}

export function getAqiMeta(aqiOrLevel: number | AQILevel): AQILevelMeta {
  const level: AQILevel =
    typeof aqiOrLevel === "number" ? getAqiLevelFromScore(aqiOrLevel) : aqiOrLevel;
  return AQI_LEVELS[level] || AQI_LEVELS.hazardous;
}
