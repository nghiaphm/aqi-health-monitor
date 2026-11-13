"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Wind } from "lucide-react";

export default function RegisterPage() {
  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-muted/20">
      <Card className="max-w-md w-full border shadow-sm text-center">
        <CardHeader className="space-y-1">
          <div className="w-10 h-10 rounded-xl bg-primary text-primary-foreground flex items-center justify-center font-bold mx-auto mb-2">
            <Wind className="w-5 h-5" />
          </div>
          <CardTitle className="text-xl">Đăng ký tài khoản</CardTitle>
          <CardDescription className="text-xs">
            Hệ thống sử dụng xác thực OIDC / Keycloak tập trung
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-xs text-muted-foreground">
            Trong phiên bản hiện tại, người dùng được đồng bộ tự động ngay khi đăng nhập lần đầu tiên qua Keycloak OIDC.
          </p>
          <Link href="/login">
            <Button className="w-full">Đi tới trang đăng nhập</Button>
          </Link>
        </CardContent>
      </Card>
    </div>
  );
}
