"use client";

import { useAuth } from "@/store/auth-store";
import { LocationPicker } from "@/components/locations/location-picker";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowLeft, MapPin, Home } from "lucide-react";

export default function LocationsPage() {
  const { user, refreshUser } = useAuth();
  const locations = user?.locations || [];

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
        <h1 className="text-2xl font-bold tracking-tight">Vị trí quan trắc</h1>
        <p className="text-xs text-muted-foreground">
          Quản lý vị trí hiện tại và nhà riêng để tra cứu chất lượng không khí chuẩn xác nhất.
        </p>
      </div>

      {/* Existing Locations */}
      {locations.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            Các vị trí đã lưu
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {locations.map((loc) => (
              <Card key={loc.id} className="p-4 border shadow-xs flex items-start gap-3">
                <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0 mt-0.5">
                  {loc.label === "home" ? <Home className="w-4 h-4" /> : <MapPin className="w-4 h-4" />}
                </div>
                <div className="space-y-0.5">
                  <span className="font-semibold text-sm">
                    {loc.label === "home" ? "Nhà riêng" : "Vị trí hiện tại"}
                  </span>
                  <p className="text-xs text-muted-foreground">
                    {loc.city || "Chưa rõ thành phố"}
                  </p>
                  <p className="text-[11px] font-mono text-muted-foreground">
                    {loc.latitude.toFixed(4)}, {loc.longitude.toFixed(4)}
                  </p>
                </div>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Add / Update Location */}
      <Card className="border shadow-xs">
        <CardHeader className="pb-3 border-b">
          <CardTitle className="text-base font-semibold">Cập nhật vị trí mới</CardTitle>
          <CardDescription className="text-xs">
            Sử dụng GPS hoặc nhập tọa độ để cập nhật vị trí hiện tại hoặc nhà riêng
          </CardDescription>
        </CardHeader>
        <CardContent className="pt-6">
          <LocationPicker
            onSuccess={() => {
              refreshUser();
            }}
          />
        </CardContent>
      </Card>
    </div>
  );
}
