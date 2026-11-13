"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/store/auth-store";
import { getCurrentAQI } from "@/lib/api/aqi";
import { CurrentAQIResponse } from "@/types/aqi";
import { ApiError } from "@/lib/api/client";
import { AqiGauge } from "@/components/aqi/aqi-gauge";
import { AqiCard } from "@/components/aqi/aqi-card";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  MapPin,
  RefreshCw,
  AlertTriangle,
  Compass,
  Wind,
  Home,
} from "lucide-react";

export default function DashboardPage() {
  const router = useRouter();
  const { user, isLoading: isAuthLoading, needsOnboarding } = useAuth();
  const [aqiData, setAqiData] = useState<CurrentAQIResponse | null>(null);
  const [selectedLocationId, setSelectedLocationId] = useState<string | undefined>(undefined);
  const [isLoadingAQI, setIsLoadingAQI] = useState<boolean>(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [isFreshnessWarning, setIsFreshnessWarning] = useState<boolean>(false);

  // If user needs onboarding, automatically redirect
  useEffect(() => {
    if (!isAuthLoading && needsOnboarding) {
      router.push("/onboarding");
    }
  }, [isAuthLoading, needsOnboarding, router]);

  const fetchAQI = useCallback(async (locId?: string) => {
    setIsLoadingAQI(true);
    setError(null);
    try {
      const data = await getCurrentAQI(locId);
      setAqiData(data);
      if (data?.reading?.measured_at) {
        const diffHours = (Date.now() - new Date(data.reading.measured_at).getTime()) / (1000 * 60 * 60);
        setIsFreshnessWarning(diffHours >= 2 && diffHours < 3);
      } else {
        setIsFreshnessWarning(false);
      }
    } catch (err: unknown) {
      setIsFreshnessWarning(false);
      if (err instanceof ApiError) {
        setError(err);
      } else {
        setError(new ApiError(500, "INTERNAL_ERROR", err instanceof Error ? err.message : "Không thể kết nối đến máy chủ."));
      }
      setAqiData(null);
    } finally {
      setIsLoadingAQI(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    if (user && !needsOnboarding) {
      async function load() {
        try {
          const data = await getCurrentAQI(selectedLocationId);
          if (!cancelled) {
            setAqiData(data);
            if (data?.reading?.measured_at) {
              const diffHours = (Date.now() - new Date(data.reading.measured_at).getTime()) / (1000 * 60 * 60);
              setIsFreshnessWarning(diffHours >= 2 && diffHours < 3);
            } else {
              setIsFreshnessWarning(false);
            }
            setError(null);
            setIsLoadingAQI(false);
          }
        } catch (err: unknown) {
          if (!cancelled) {
            setIsFreshnessWarning(false);
            if (err instanceof ApiError) {
              setError(err);
            } else {
              setError(
                new ApiError(
                  500,
                  "INTERNAL_ERROR",
                  err instanceof Error ? err.message : "Không thể kết nối đến máy chủ."
                )
              );
            }
            setAqiData(null);
            setIsLoadingAQI(false);
          }
        }
      }

      load();
    }

    return () => {
      cancelled = true;
    };
  }, [user, needsOnboarding, selectedLocationId]);

  if (isAuthLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <Wind className="w-8 h-8 text-primary animate-spin" />
          <span className="text-sm text-muted-foreground">Đang tải hồ sơ...</span>
        </div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <Card className="max-w-md w-full text-center p-6 space-y-4">
          <Wind className="w-12 h-12 text-primary mx-auto" />
          <h2 className="text-xl font-bold">Chào mừng đến với AQI Health Monitor</h2>
          <p className="text-sm text-muted-foreground">
            Vui lòng đăng nhập hoặc cấu hình token xác thực để bắt đầu theo dõi chất lượng không khí.
          </p>
          <div className="pt-2 flex flex-col gap-2">
            <Link href="/login">
              <Button className="w-full">Đăng nhập tài khoản</Button>
            </Link>
          </div>
        </Card>
      </div>
    );
  }

  const userLocations = user.locations || [];

  return (
    <div className="min-h-screen bg-muted/10 pb-16">
      {/* Top Navbar */}
      <header className="sticky top-0 z-20 border-b bg-background/80 backdrop-blur-md">
        <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-primary text-primary-foreground flex items-center justify-center font-bold">
              <Wind className="w-4 h-4" />
            </div>
            <span className="font-bold text-base tracking-tight hidden sm:inline">AQI Health Monitor</span>
          </div>

          <div className="flex items-center gap-2">
            {/* Location Switcher */}
            {userLocations.length > 0 && (
              <div className="flex items-center gap-1 bg-muted/60 p-1 rounded-lg text-xs">
                {userLocations.map((loc) => {
                  const isCurrent = (selectedLocationId === undefined && loc.label === "current") || selectedLocationId === loc.id;
                  return (
                    <button
                      key={loc.id}
                      onClick={() => {
                        setSelectedLocationId(loc.id);
                      }}
                      className={`flex items-center gap-1 px-2.5 py-1 rounded-md font-medium transition-all ${
                        isCurrent
                          ? "bg-background text-foreground shadow-xs"
                          : "text-muted-foreground hover:text-foreground"
                      }`}
                    >
                      {loc.label === "home" ? <Home className="w-3 h-3" /> : <MapPin className="w-3 h-3" />}
                      <span>{loc.label === "home" ? "Nhà riêng" : "Hiện tại"}</span>
                    </button>
                  );
                })}
              </div>
            )}

            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchAQI(selectedLocationId)}
              disabled={isLoadingAQI}
              className="h-8 gap-1.5"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isLoadingAQI ? "animate-spin" : ""}`} />
              <span className="hidden sm:inline">Làm mới</span>
            </Button>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="max-w-3xl mx-auto px-4 pt-6 space-y-6">
        {/* Warning Banner: Freshness >= 2h */}
        {isFreshnessWarning && (
          <div className="flex items-start gap-3 p-3.5 rounded-xl border border-yellow-500/30 bg-yellow-500/10 text-yellow-900 dark:text-yellow-200 text-xs">
            <AlertTriangle className="w-4 h-4 shrink-0 text-yellow-600 dark:text-yellow-400 mt-0.5" />
            <div className="space-y-0.5">
              <p className="font-semibold">Dữ liệu có thể chưa hoàn toàn mới</p>
              <p className="opacity-90">
                Bản ghi đo từ trạm quan trắc đã cách đây hơn 2 giờ. Số liệu có thể chưa cập nhật kịp biến động không khí gần nhất.
              </p>
            </div>
          </div>
        )}

        {/* Loading State */}
        {isLoadingAQI && !aqiData && (
          <Card className="p-12 text-center border shadow-xs space-y-4">
            <div className="w-12 h-12 rounded-full bg-primary/10 text-primary flex items-center justify-center mx-auto animate-spin">
              <RefreshCw className="w-6 h-6" />
            </div>
            <p className="text-sm font-medium text-muted-foreground">Đang cập nhật chỉ số chất lượng không khí...</p>
          </Card>
        )}

        {/* Error States */}
        {!isLoadingAQI && error && (
          <Card className="border shadow-xs p-8 text-center space-y-4">
            <div className="w-12 h-12 rounded-full bg-destructive/10 text-destructive flex items-center justify-center mx-auto">
              <AlertTriangle className="w-6 h-6" />
            </div>

            {error.code === "LOCATION_NOT_FOUND" && (
              <div className="space-y-3">
                <h3 className="font-bold text-base text-foreground">Bạn chưa thiết lập vị trí theo dõi</h3>
                <p className="text-xs text-muted-foreground max-w-sm mx-auto">
                  Hệ thống cần tọa độ của bạn để xác định trạm quan trắc gần nhất và cung cấp chỉ số AQI chính xác.
                </p>
                <div className="pt-2">
                  <Link href="/onboarding">
                    <Button size="sm" className="gap-1.5">
                      <Compass className="w-4 h-4" />
                      Thiết lập vị trí ngay
                    </Button>
                  </Link>
                </div>
              </div>
            )}

            {error.code === "NO_STATION_COVERAGE" && (
              <div className="space-y-2">
                <h3 className="font-bold text-base text-foreground">Ngoài vùng phủ sóng quan trắc</h3>
                <p className="text-xs text-muted-foreground max-w-sm mx-auto">
                  Hiện tại không có trạm quan trắc không khí nào hoạt động trong bán kính 20 km tính từ vị trí của bạn.
                </p>
              </div>
            )}

            {error.code === "NO_DATA" && (
              <div className="space-y-2">
                <h3 className="font-bold text-base text-foreground">Trạm đo chưa có dữ liệu</h3>
                <p className="text-xs text-muted-foreground max-w-sm mx-auto">
                  Trạm quan trắc gần bạn đang trong quá trình bảo trì hoặc chưa có số liệu đo mới. Vui lòng quay lại sau.
                </p>
              </div>
            )}

            {error.code === "DATA_STALE" && (
              <div className="space-y-2">
                <h3 className="font-bold text-base text-foreground">Dữ liệu đo quá hạn (&gt; 3 giờ)</h3>
                <p className="text-xs text-muted-foreground max-w-sm mx-auto">
                  Dữ liệu quan trắc gần nhất đã vượt quá 3 giờ so với hiện tại, không đảm bảo độ tin cậy để hiển thị.
                </p>
              </div>
            )}

            {error.code !== "LOCATION_NOT_FOUND" &&
              error.code !== "NO_STATION_COVERAGE" &&
              error.code !== "NO_DATA" &&
              error.code !== "DATA_STALE" && (
                <div className="space-y-3">
                  <h3 className="font-bold text-base text-foreground">Không thể tải dữ liệu</h3>
                  <p className="text-xs text-muted-foreground max-w-sm mx-auto">{error.message}</p>
                  <Button size="sm" variant="outline" onClick={() => fetchAQI(selectedLocationId)}>
                    Thử lại
                  </Button>
                </div>
              )}
          </Card>
        )}

        {/* Success AQI Display */}
        {aqiData && (
          <div className="space-y-6">
            {/* Center Gauge Card */}
            <Card className="border shadow-xs bg-card overflow-hidden">
              <AqiGauge aqi={aqiData.reading.aqi} level={aqiData.reading.level} />
            </Card>

            {/* Detailed Readings, Pollutants and Advice */}
            <AqiCard data={aqiData} />
          </div>
        )}
      </main>
    </div>
  );
}
