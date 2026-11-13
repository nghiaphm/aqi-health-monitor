"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ArrowLeft, LineChart } from "lucide-react";

export default function HistoryPage() {
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
        <h1 className="text-2xl font-bold tracking-tight">Lịch sử chỉ số AQI</h1>
        <p className="text-xs text-muted-foreground">
          Theo dõi diễn biến chất lượng không khí theo thời gian 7 ngày hoặc 30 ngày.
        </p>
      </div>

      <Card className="p-12 text-center border shadow-xs space-y-3">
        <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mx-auto text-muted-foreground">
          <LineChart className="w-6 h-6" />
        </div>
        <h3 className="font-semibold text-sm">Tính năng đang được phát triển</h3>
        <p className="text-xs text-muted-foreground max-w-sm mx-auto">
          Biểu đồ xu hướng và phân tích lịch sử AQI sẽ được tích hợp trong các giai đoạn tiếp theo.
        </p>
      </Card>
    </div>
  );
}
