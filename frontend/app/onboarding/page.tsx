"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/store/auth-store";
import { HealthProfileForm } from "@/components/health-profile/health-profile-form";
import { LocationPicker } from "@/components/locations/location-picker";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Check, Wind } from "lucide-react";

export default function OnboardingPage() {
  const router = useRouter();
  const { user, isLoading } = useAuth();
  const [currentStep, setCurrentStep] = useState<1 | 2>(1);

  // If user already completed both health profile and location, route to dashboard
  useEffect(() => {
    if (!isLoading && user?.health_profile && user.locations && user.locations.length > 0) {
      router.push("/dashboard");
    }
  }, [user, isLoading, router]);

  const handleHealthProfileSuccess = () => {
    setCurrentStep(2);
  };

  const handleLocationSuccess = () => {
    router.push("/dashboard");
  };

  const handleSkipLocation = () => {
    router.push("/dashboard");
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <Wind className="w-8 h-8 text-primary animate-bounce" />
          <span className="text-sm text-muted-foreground">Đang tải thông tin...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col justify-center py-12 sm:px-6 lg:px-8 bg-muted/20">
      <div className="sm:mx-auto sm:w-full sm:max-w-xl">
        <div className="flex justify-center items-center gap-2 mb-4">
          <div className="w-9 h-9 rounded-xl bg-primary text-primary-foreground flex items-center justify-center font-bold">
            <Wind className="w-5 h-5" />
          </div>
          <span className="font-bold text-xl tracking-tight">AQI Health Monitor</span>
        </div>

        {/* Stepper Header */}
        <div className="flex items-center justify-center gap-4 mb-6">
          <div className="flex items-center gap-2">
            <div
              className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold ${
                currentStep === 1
                  ? "bg-primary text-primary-foreground"
                  : "bg-primary/20 text-primary"
              }`}
            >
              {currentStep > 1 ? <Check className="w-4 h-4" /> : "1"}
            </div>
            <span
              className={`text-xs font-medium ${
                currentStep === 1 ? "text-foreground" : "text-muted-foreground"
              }`}
            >
              Hồ sơ sức khỏe
            </span>
          </div>

          <div className="w-8 h-px bg-border" />

          <div className="flex items-center gap-2">
            <div
              className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold ${
                currentStep === 2
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground"
              }`}
            >
              2
            </div>
            <span
              className={`text-xs font-medium ${
                currentStep === 2 ? "text-foreground" : "text-muted-foreground"
              }`}
            >
              Vị trí quan trắc
            </span>
          </div>
        </div>

        {/* Card Body */}
        <Card className="shadow-sm border">
          <CardHeader className="text-center pb-4">
            <CardTitle className="text-xl">
              {currentStep === 1
                ? "Thiết lập Hồ sơ Sức khỏe Cá nhân"
                : "Chọn vị trí để theo dõi chất lượng không khí"}
            </CardTitle>
            <CardDescription className="text-xs">
              {currentStep === 1
                ? "Giúp hệ thống tự động xác định ngưỡng AQI cảnh báo riêng cho bạn."
                : "Kết nối trạm quan trắc không khí gần nhất để cập nhật dữ liệu liên tục."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {currentStep === 1 ? (
              <HealthProfileForm
                onSuccess={handleHealthProfileSuccess}
                initialCondition={user?.health_profile?.condition_type}
                initialAgeGroup={user?.health_profile?.age_group}
                initialSensitivity={user?.health_profile?.sensitivity_level}
              />
            ) : (
              <LocationPicker
                onSuccess={handleLocationSuccess}
                onSkip={handleSkipLocation}
              />
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
