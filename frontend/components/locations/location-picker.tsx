"use client";

import { useState } from "react";
import { UpsertLocationResponse, LocationLabel } from "@/types/location";
import { upsertLocation } from "@/lib/api/locations";
import { useAuth } from "@/store/auth-store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { MapPin, Navigation, AlertCircle, ChevronRight } from "lucide-react";

interface LocationPickerProps {
  onSuccess?: (res: UpsertLocationResponse) => void;
  onSkip?: () => void;
  defaultLabel?: LocationLabel;
}

export function LocationPicker({
  onSuccess,
  onSkip,
  defaultLabel = "current",
}: LocationPickerProps) {
  const { refreshUser } = useAuth();
  const [isGettingGps, setIsGettingGps] = useState(false);
  const [isSubmittingManual, setIsSubmittingManual] = useState(false);
  const [showManualForm, setShowManualForm] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Manual form fields
  const [label, setLabel] = useState<LocationLabel>(defaultLabel);
  const [latitude, setLatitude] = useState<string>("21.028511"); // Mặc định Hà Nội
  const [longitude, setLongitude] = useState<string>("105.854444");
  const [city, setCity] = useState<string>("Hà Nội");

  const handleUseGps = () => {
    setErrorMsg(null);

    if (!navigator.geolocation) {
      setErrorMsg("Trình duyệt không hỗ trợ định vị GPS. Vui lòng nhập vị trí thủ công.");
      setShowManualForm(true);
      return;
    }

    setIsGettingGps(true);
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        try {
          const lat = parseFloat(pos.coords.latitude.toFixed(6));
          const lng = parseFloat(pos.coords.longitude.toFixed(6));

          const res = await upsertLocation({
            label: "current",
            latitude: lat,
            longitude: lng,
            city: "Vị trí GPS",
          });

          await refreshUser();
          if (onSuccess) onSuccess(res);
        } catch (err: unknown) {
          setErrorMsg(err instanceof Error ? err.message : "Không thể lưu vị trí GPS lên máy chủ.");
        } finally {
          setIsGettingGps(false);
        }
      },
      (err) => {
        setIsGettingGps(false);
        if (err.code === err.PERMISSION_DENIED) {
          setErrorMsg("Bạn đã từ chối quyền truy cập vị trí. Hãy nhập tọa độ hoặc thành phố thủ công.");
        } else {
          setErrorMsg("Không thể xác định vị trí GPS. Vui lòng nhập thủ công.");
        }
        setShowManualForm(true);
      },
      { timeout: 10000, enableHighAccuracy: true }
    );
  };

  const handleManualSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    const latNum = parseFloat(latitude);
    const lngNum = parseFloat(longitude);

    if (isNaN(latNum) || latNum < -90 || latNum > 90) {
      setErrorMsg("Vĩ độ (latitude) không hợp lệ, phải nằm trong khoảng [-90, 90].");
      return;
    }
    if (isNaN(lngNum) || lngNum < -180 || lngNum > 180) {
      setErrorMsg("Kinh độ (longitude) không hợp lệ, phải nằm trong khoảng [-180, 180].");
      return;
    }

    setIsSubmittingManual(true);
    try {
      const res = await upsertLocation({
        label,
        latitude: latNum,
        longitude: lngNum,
        city: city.trim() || null,
      });

      await refreshUser();
      if (onSuccess) onSuccess(res);
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : "Không thể lưu vị trí thủ công. Vui lòng thử lại.");
    } finally {
      setIsSubmittingManual(false);
    }
  };

  return (
    <div className="space-y-6">
      {errorMsg && (
        <div className="flex items-center gap-2 p-3 text-sm rounded-lg bg-destructive/10 text-destructive border border-destructive/20">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{errorMsg}</span>
        </div>
      )}

      {/* Primary: GPS Button */}
      <div className="p-6 rounded-2xl border bg-card text-center space-y-4">
        <div className="w-12 h-12 rounded-full bg-primary/10 text-primary flex items-center justify-center mx-auto">
          <Navigation className="w-6 h-6 animate-pulse" />
        </div>
        <div className="space-y-1">
          <h3 className="font-semibold text-base">Định vị tự động qua GPS</h3>
          <p className="text-xs text-muted-foreground max-w-sm mx-auto">
            Hệ thống sẽ lấy tọa độ hiện tại của bạn để tìm trạm đo chất lượng không khí gần nhất trong bán kính 20 km.
          </p>
        </div>
        <Button
          type="button"
          onClick={handleUseGps}
          disabled={isGettingGps}
          className="w-full sm:w-auto px-6 h-11 text-sm font-medium"
        >
          {isGettingGps ? "Đang lấy tọa độ GPS..." : "Dùng vị trí hiện tại của tôi"}
        </Button>
      </div>

      {/* Secondary: Manual Toggle */}
      <div className="text-center">
        {!showManualForm ? (
          <button
            type="button"
            onClick={() => setShowManualForm(true)}
            className="text-xs text-muted-foreground hover:text-foreground underline underline-offset-4"
          >
            Hoặc nhập tọa độ / địa chỉ thủ công
          </button>
        ) : (
          <form onSubmit={handleManualSubmit} className="p-5 rounded-2xl border bg-card text-left space-y-4">
            <div className="flex items-center justify-between pb-2 border-b">
              <span className="font-semibold text-sm flex items-center gap-2">
                <MapPin className="w-4 h-4 text-primary" />
                Nhập vị trí thủ công
              </span>
              <button
                type="button"
                onClick={() => setShowManualForm(false)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Ẩn bớt
              </button>
            </div>

            <div className="space-y-2">
              <label className="text-xs font-medium">Loại vị trí</label>
              <div className="flex gap-4">
                <label className="flex items-center gap-2 text-xs cursor-pointer">
                  <input
                    type="radio"
                    name="location_label"
                    value="current"
                    checked={label === "current"}
                    onChange={() => setLabel("current")}
                  />
                  <span>Vị trí hiện tại (current)</span>
                </label>
                <label className="flex items-center gap-2 text-xs cursor-pointer">
                  <input
                    type="radio"
                    name="location_label"
                    value="home"
                    checked={label === "home"}
                    onChange={() => setLabel("home")}
                  />
                  <span>Nhà riêng (home)</span>
                </label>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs font-medium">Vĩ độ (Latitude)</label>
                <Input
                  value={latitude}
                  onChange={(e) => setLatitude(e.target.value)}
                  placeholder="21.028511"
                  className="h-9 text-xs"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-medium">Kinh độ (Longitude)</label>
                <Input
                  value={longitude}
                  onChange={(e) => setLongitude(e.target.value)}
                  placeholder="105.854444"
                  className="h-9 text-xs"
                />
              </div>
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium">Tên thành phố / Khu vực (tùy chọn)</label>
              <Input
                value={city}
                onChange={(e) => setCity(e.target.value)}
                placeholder="Hà Nội hoặc TP. Hồ Chí Minh"
                className="h-9 text-xs"
              />
            </div>

            <Button type="submit" disabled={isSubmittingManual} className="w-full h-9 text-xs">
              {isSubmittingManual ? "Đang lưu..." : "Xác nhận lưu vị trí"}
            </Button>
          </form>
        )}
      </div>

      {/* Tertiary: Skip Option */}
      {onSkip && (
        <div className="pt-2 text-center border-t">
          <button
            type="button"
            onClick={onSkip}
            className="text-xs text-muted-foreground hover:text-foreground inline-flex items-center gap-1"
          >
            Bỏ qua bước này, bổ sung vị trí sau
            <ChevronRight className="w-3.5 h-3.5" />
          </button>
        </div>
      )}
    </div>
  );
}
