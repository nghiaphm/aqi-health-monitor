"use client";

import { useAuth } from "@/store/auth-store";
import { HealthProfileForm } from "@/components/health-profile/health-profile-form";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Bell } from "lucide-react";

export default function ProfilePage() {
  const { user, refreshUser } = useAuth();

  return (
    <div className="max-w-2xl mx-auto px-4 py-8 space-y-6">
      <div className="flex items-center justify-between">
        <Link href="/dashboard">
          <Button variant="ghost" size="sm" className="gap-1.5 text-xs">
            <ArrowLeft className="w-3.5 h-3.5" />
            Về Dashboard
          </Button>
        </Link>
        <span className="text-xs text-muted-foreground">{user?.email}</span>
      </div>

      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Hồ sơ sức khỏe</h1>
        <p className="text-xs text-muted-foreground">
          Cập nhật tình trạng bệnh lý và độ tuổi để hệ thống điều chỉnh ngưỡng cảnh báo AQI phù hợp.
        </p>
      </div>

      {user?.alert_threshold && (
        <Card className="border shadow-xs bg-primary/5 border-primary/20">
          <CardContent className="p-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
                <Bell className="w-4 h-4" />
              </div>
              <div>
                <span className="text-xs font-semibold text-foreground">Ngưỡng cảnh báo hiện tại</span>
                <p className="text-xs text-muted-foreground">
                  Hệ thống sẽ gửi cảnh báo khi AQI vượt ngưỡng này
                </p>
              </div>
            </div>
            <span className="text-2xl font-black text-primary">
              {user.alert_threshold.threshold_aqi}
            </span>
          </CardContent>
        </Card>
      )}

      <Card className="border shadow-xs">
        <CardHeader className="pb-3 border-b">
          <CardTitle className="text-base font-semibold">Thông tin chi tiết</CardTitle>
          <CardDescription className="text-xs">
            Mọi thay đổi sẽ cập nhật trực tiếp ngưỡng cảnh báo
          </CardDescription>
        </CardHeader>
        <CardContent className="pt-6">
          <HealthProfileForm
            initialCondition={user?.health_profile?.condition_type}
            initialAgeGroup={user?.health_profile?.age_group}
            initialSensitivity={user?.health_profile?.sensitivity_level}
            submitButtonText="Lưu thay đổi"
            onSuccess={() => {
              refreshUser();
            }}
          />
        </CardContent>
      </Card>
    </div>
  );
}
