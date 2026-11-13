"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/store/auth-store";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Wind, ShieldCheck, MapPin, Bell, ArrowRight, KeyRound } from "lucide-react";

export default function Home() {
  const router = useRouter();
  const { user, loginWithToken, logout } = useAuth();
  const [inputToken, setInputToken] = useState("");
  const [showDevTokenInput, setShowDevTokenInput] = useState(false);

  const handleApplyToken = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputToken.trim()) return;
    await loginWithToken(inputToken.trim());
    router.push("/dashboard");
  };

  return (
    <div className="min-h-screen flex flex-col justify-between bg-gradient-to-b from-background via-muted/10 to-muted/30">
      {/* Navigation */}
      <header className="border-b bg-background/80 backdrop-blur-md">
        <div className="max-w-5xl mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-xl bg-primary text-primary-foreground flex items-center justify-center font-bold shadow-xs">
              <Wind className="w-5 h-5" />
            </div>
            <span className="font-bold text-lg tracking-tight">AQI Health Monitor</span>
          </div>

          <div className="flex items-center gap-3">
            {user ? (
              <div className="flex items-center gap-3">
                <span className="text-xs text-muted-foreground hidden sm:inline">
                  {user.email || user.full_name || "Đã đăng nhập"}
                </span>
                <Link href="/dashboard">
                  <Button size="sm">Vào Dashboard</Button>
                </Link>
                <Button size="sm" variant="outline" onClick={logout}>
                  Đăng xuất
                </Button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setShowDevTokenInput(!showDevTokenInput)}
                  className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1 px-2 py-1"
                >
                  <KeyRound className="w-3.5 h-3.5" />
                  Dev Token
                </button>
                <Link href="/dashboard">
                  <Button size="sm" className="gap-1.5">
                    Bắt đầu ngay
                    <ArrowRight className="w-3.5 h-3.5" />
                  </Button>
                </Link>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <main className="max-w-4xl mx-auto px-4 py-16 text-center space-y-8">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-primary/20 bg-primary/5 text-primary text-xs font-semibold">
          <ShieldCheck className="w-3.5 h-3.5" />
          <span>Theo dõi không khí thông minh cho sức khỏe gia đình</span>
        </div>

        <h1 className="text-4xl sm:text-5xl font-black tracking-tight leading-tight max-w-2xl mx-auto text-foreground">
          Bảo vệ hô hấp với cảnh báo chất lượng không khí cá nhân hóa
        </h1>

        <p className="text-base text-muted-foreground max-w-xl mx-auto leading-relaxed">
          Tự động điều chỉnh ngưỡng cảnh báo AQI theo bệnh lý nền (hen suyễn, viêm mũi) và độ tuổi của bạn,
          kết nối dữ liệu thời gian thực từ trạm quan trắc gần nhất.
        </p>

        {/* Dev Token Box (để test khi Keycloak realm đang defer) */}
        {showDevTokenInput && (
          <Card className="max-w-md mx-auto p-4 border text-left shadow-sm bg-card">
            <CardHeader className="p-0 pb-3">
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <KeyRound className="w-4 h-4 text-primary" />
                Cấu hình JWT Token (Môi trường Dev / Test)
              </CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <form onSubmit={handleApplyToken} className="space-y-3">
                <Input
                  value={inputToken}
                  onChange={(e) => setInputToken(e.target.value)}
                  placeholder="Dán token JWT để gọi API backend..."
                  className="text-xs h-9"
                />
                <Button type="submit" size="sm" className="w-full h-8 text-xs">
                  Áp dụng & Đi tới Dashboard
                </Button>
              </form>
            </CardContent>
          </Card>
        )}

        <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-4">
          <Link href="/dashboard" className="w-full sm:w-auto">
            <Button size="lg" className="w-full sm:w-auto px-8 text-base h-12 gap-2 shadow-sm">
              Trải nghiệm Dashboard
              <ArrowRight className="w-4 h-4" />
            </Button>
          </Link>
          <Link href="/onboarding" className="w-full sm:w-auto">
            <Button size="lg" variant="outline" className="w-full sm:w-auto px-8 text-base h-12">
              Thiết lập hồ sơ sức khỏe
            </Button>
          </Link>
        </div>

        {/* Feature Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-12 text-left">
          <div className="p-5 rounded-2xl border bg-card/60 backdrop-blur-xs space-y-2">
            <div className="w-9 h-9 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-sm">Hồ sơ cá nhân hóa</h3>
            <p className="text-xs text-muted-foreground leading-normal">
              Ma trận 18 tổ hợp ngưỡng AQI tự động hạ thấp với nhóm nhạy cảm và trẻ em.
            </p>
          </div>

          <div className="p-5 rounded-2xl border bg-card/60 backdrop-blur-xs space-y-2">
            <div className="w-9 h-9 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
              <MapPin className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-sm">Trạm đo trong 20 km</h3>
            <p className="text-xs text-muted-foreground leading-normal">
              Thuật toán PostGIS tự động tìm kiếm trạm hoạt động gần tọa độ của bạn nhất.
            </p>
          </div>

          <div className="p-5 rounded-2xl border bg-card/60 backdrop-blur-xs space-y-2">
            <div className="w-9 h-9 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
              <Bell className="w-5 h-5" />
            </div>
            <h3 className="font-semibold text-sm">Khuyến nghị thực tế</h3>
            <p className="text-xs text-muted-foreground leading-normal">
              Chỉ dẫn hành động phòng hộ chi tiết theo từng khoảng mức AQI chuẩn EPA.
            </p>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t py-6 text-center text-xs text-muted-foreground">
        <p>AQI Health Monitor · Dự án theo dõi chất lượng không khí & bảo vệ sức khỏe</p>
      </footer>
    </div>
  );
}
