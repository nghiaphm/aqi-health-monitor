"use client";

import { useState } from "react";
import { usePushSubscription } from "@/lib/hooks/use-push-subscription";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  AlertCircle,
  BellOff,
  BellRing,
  CheckCircle2,
  Info,
  Loader2,
} from "lucide-react";

export function PushSubscriptionCard() {
  const {
    isSupported,
    isVapidConfigured,
    permission,
    isSubscribed,
    isLoading,
    isSubmitting,
    error,
    subscribe,
    unsubscribe,
  } = usePushSubscription();
  const [feedback, setFeedback] = useState<string | null>(null);

  const handleToggle = async () => {
    setFeedback(null);
    if (isSubscribed) {
      const ok = await unsubscribe();
      if (ok) {
        setFeedback("Đã tắt thông báo đẩy trên trình duyệt này.");
      }
      return;
    }

    const ok = await subscribe();
    if (ok) {
      setFeedback("Đã bật thông báo đẩy trên trình duyệt này.");
    }
  };

  const renderStatusBadge = () => {
    if (isLoading) {
      return <Badge variant="secondary">Đang kiểm tra...</Badge>;
    }
    if (!isSupported || permission === "unsupported") {
      return <Badge variant="secondary">Không hỗ trợ</Badge>;
    }
    if (permission === "denied") {
      return <Badge variant="destructive">Bị chặn</Badge>;
    }
    if (isSubscribed) {
      return (
        <Badge className="bg-emerald-600 hover:bg-emerald-600 text-white">
          Đang bật
        </Badge>
      );
    }
    return <Badge variant="outline">Chưa bật</Badge>;
  };

  return (
    <Card className="border shadow-xs">
      <CardHeader className="pb-3 border-b">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0">
              {isSubscribed ? (
                <BellRing className="w-4 h-4" />
              ) : (
                <BellOff className="w-4 h-4" />
              )}
            </div>
            <div>
              <CardTitle className="text-base font-semibold">
                Thông báo đẩy trên trình duyệt (Web Push)
              </CardTitle>
              <CardDescription className="text-xs">
                Nhận cảnh báo tức thì ngay cả khi bạn không mở ứng dụng.
              </CardDescription>
            </div>
          </div>
          {renderStatusBadge()}
        </div>
      </CardHeader>

      <CardContent className="pt-4 space-y-3">
        {!isLoading && (!isSupported || permission === "unsupported") && (
          <div className="flex items-start gap-2.5 p-3 rounded-lg bg-muted text-muted-foreground text-xs">
            <Info className="w-4 h-4 shrink-0 mt-0.5" />
            <div className="space-y-1">
              <p className="font-medium text-foreground">
                Trình duyệt không hỗ trợ Web Push
              </p>
              <p>
                Trình duyệt hoặc thiết bị hiện tại không hỗ trợ Push API /
                Service Worker. Hệ thống vẫn tự động gửi cảnh báo qua Email khi
                AQI vượt ngưỡng của bạn.
              </p>
            </div>
          </div>
        )}

        {!isLoading && isSupported && permission === "denied" && (
          <div className="flex items-start gap-2.5 p-3 rounded-lg bg-amber-500/10 text-amber-800 dark:text-amber-200 border border-amber-500/20 text-xs">
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
            <div className="space-y-1">
              <p className="font-medium">
                Quyền thông báo đang bị chặn trên trình duyệt
              </p>
              <p>
                Để bật lại: nhấn biểu tượng khóa / cài đặt trang web cạnh thanh
                địa chỉ trình duyệt → mục <strong>Thông báo (Notifications)</strong>{" "}
                → chọn <strong>Cho phép (Allow)</strong> rồi tải lại trang.
              </p>
            </div>
          </div>
        )}

        {error && (
          <div className="flex items-start gap-2 p-3 text-xs rounded-lg bg-destructive/10 text-destructive border border-destructive/20">
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        {feedback && !error && (
          <div className="flex items-start gap-2 p-3 text-xs rounded-lg bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border border-emerald-500/20">
            <CheckCircle2 className="w-4 h-4 shrink-0 mt-0.5" />
            <span>{feedback}</span>
          </div>
        )}

        {isSupported && permission !== "denied" && (
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 pt-1">
            <div className="text-xs text-muted-foreground">
              {isSubscribed ? (
                <span>
                  Trình duyệt này đã được đăng ký nhận cảnh báo AQI vượt ngưỡng.
                </span>
              ) : (
                <span>
                  Bật để đồng bộ thiết bị này với máy chủ cảnh báo.
                  {!isVapidConfigured &&
                    " (Cần biến NEXT_PUBLIC_VAPID_PUBLIC_KEY để kích hoạt.)"}
                </span>
              )}
            </div>

            <Button
              type="button"
              variant={isSubscribed ? "outline" : "default"}
              size="sm"
              disabled={isLoading || isSubmitting || (!isSubscribed && !isVapidConfigured)}
              onClick={() => void handleToggle()}
              className="shrink-0 gap-2"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  Đang xử lý...
                </>
              ) : isSubscribed ? (
                <>
                  <BellOff className="w-3.5 h-3.5" />
                  Tắt thông báo đẩy
                </>
              ) : (
                <>
                  <BellRing className="w-3.5 h-3.5" />
                  Bật thông báo đẩy
                </>
              )}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
