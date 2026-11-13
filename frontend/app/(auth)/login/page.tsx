"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/store/auth-store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Wind, AlertCircle } from "lucide-react";

export default function LoginPage() {
  const router = useRouter();
  const { loginWithToken } = useAuth();
  const [tokenInput, setTokenInput] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tokenInput.trim()) {
      setErrorMsg("Vui lòng nhập Bearer JWT Token.");
      return;
    }

    setIsLoading(true);
    setErrorMsg(null);
    try {
      await loginWithToken(tokenInput.trim());
      router.push("/dashboard");
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : "Đăng nhập thất bại. Vui lòng kiểm tra lại token.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-muted/20">
      <Card className="max-w-md w-full border shadow-sm">
        <CardHeader className="text-center space-y-1">
          <div className="w-10 h-10 rounded-xl bg-primary text-primary-foreground flex items-center justify-center font-bold mx-auto mb-2">
            <Wind className="w-5 h-5" />
          </div>
          <CardTitle className="text-xl">Đăng nhập tài khoản</CardTitle>
          <CardDescription className="text-xs">
            Nhập Bearer Token để xác thực với hệ thống AQI Health Monitor
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {errorMsg && (
              <div className="flex items-center gap-2 p-3 text-xs rounded-lg bg-destructive/10 text-destructive border border-destructive/20">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{errorMsg}</span>
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">JWT Bearer Token</label>
              <Input
                value={tokenInput}
                onChange={(e) => setTokenInput(e.target.value)}
                placeholder="eyJhbGciOiJSUzI1NiIs..."
                className="text-xs h-9 font-mono"
              />
              <span className="text-[11px] text-muted-foreground">
                (Trong giai đoạn Tracer Bullet, dùng token hợp lệ sinh từ OIDC harness hoặc Keycloak)
              </span>
            </div>

            <Button type="submit" disabled={isLoading} className="w-full h-10 text-sm">
              {isLoading ? "Đang xác thực..." : "Xác nhận & Đăng nhập"}
            </Button>

            <div className="text-center pt-2">
              <Link href="/" className="text-xs text-muted-foreground hover:text-foreground">
                ← Quay lại trang chủ
              </Link>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
