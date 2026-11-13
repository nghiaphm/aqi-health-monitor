"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ArrowLeft, Bell } from "lucide-react";

export default function AlertsPage() {
  return (
    <div className="max-w-2xl mx-auto px-4 py-8 space-y-6">
      <div className="flex items-center justify-between">
        <Link href="/dashboard">
          <Button variant="ghost" size="sm" className="gap-1.5 text-xs">
            <ArrowLeft className="w-3.5 h-3.5" />
            Về Dashboard
          </Button>
        </Link>
      </div>

      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Lịch sử cảnh báo</h1>
        <p className="text-xs text-muted-foreground">
          Xem lại các thông báo chất lượng không khí vượt ngưỡng bảo vệ của bạn.
        </p>
      </div>

      <Card className="p-12 text-center border shadow-xs space-y-3">
        <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mx-auto text-muted-foreground">
          <Bell className="w-6 h-6" />
        </div>
        <h3 className="font-semibold text-sm">Chưa có cảnh báo nào</h3>
        <p className="text-xs text-muted-foreground max-w-sm mx-auto">
          Hệ thống sẽ ghi nhận và gửi thông báo khi chỉ số AQI thực tế vượt ngưỡng an toàn của bạn.
        </p>
      </Card>
    </div>
  );
}
