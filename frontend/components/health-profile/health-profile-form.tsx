"use client";

import { useState } from "react";
import {
  ConditionType,
  AgeGroup,
  SensitivityLevel,
  UpsertHealthProfileResponse,
} from "@/types/health-profile";
import { upsertHealthProfile } from "@/lib/api/health-profile";
import { useAuth } from "@/store/auth-store";
import { Button } from "@/components/ui/button";
import { AlertCircle, CheckCircle2 } from "lucide-react";

interface HealthProfileFormProps {
  onSuccess?: (res: UpsertHealthProfileResponse) => void;
  initialCondition?: ConditionType;
  initialAgeGroup?: AgeGroup;
  initialSensitivity?: SensitivityLevel;
  submitButtonText?: string;
}

export function HealthProfileForm({
  onSuccess,
  initialCondition = "none",
  initialAgeGroup = "adult",
  initialSensitivity = "normal",
  submitButtonText = "Lưu & Tiếp tục",
}: HealthProfileFormProps) {
  const { refreshUser } = useAuth();
  const [condition, setCondition] = useState<ConditionType>(initialCondition);
  const [ageGroup, setAgeGroup] = useState<AgeGroup>(initialAgeGroup);
  const [sensitivity, setSensitivity] = useState<SensitivityLevel>(initialSensitivity);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setErrorMsg(null);

    try {
      const res = await upsertHealthProfile({
        condition_type: condition,
        age_group: ageGroup,
        sensitivity_level: sensitivity,
      });

      await refreshUser();
      if (onSuccess) {
        onSuccess(res);
      }
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : "Không thể lưu hồ sơ sức khỏe. Vui lòng thử lại.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {errorMsg && (
        <div className="flex items-center gap-2 p-3 text-sm rounded-lg bg-destructive/10 text-destructive border border-destructive/20">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{errorMsg}</span>
        </div>
      )}

      {/* 1. Condition Type */}
      <div className="space-y-3">
        <label className="text-sm font-semibold">Tình trạng bệnh lý nền</label>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {[
            { value: "none", label: "Không có", desc: "Không có tiền sử bệnh hô hấp" },
            { value: "asthma", label: "Hen suyễn", desc: "Khó thở, thở khò khè, hen phế quản" },
            { value: "allergic_rhinitis", label: "Viêm mũi dị ứng", desc: "Dễ kích ứng mũi xoang, hắt hơi" },
          ].map((item) => {
            const isSelected = condition === item.value;
            return (
              <button
                type="button"
                key={item.value}
                onClick={() => setCondition(item.value as ConditionType)}
                className={`flex flex-col text-left p-3.5 rounded-xl border transition-all ${
                  isSelected
                    ? "border-primary bg-primary/5 ring-2 ring-primary/20 text-foreground"
                    : "border-border hover:bg-muted/50 text-muted-foreground hover:text-foreground"
                }`}
              >
                <div className="flex items-center justify-between w-full">
                  <span className="font-medium text-sm text-foreground">{item.label}</span>
                  {isSelected && <CheckCircle2 className="w-4 h-4 text-primary" />}
                </div>
                <span className="text-xs mt-1 text-muted-foreground">{item.desc}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* 2. Age Group */}
      <div className="space-y-3">
        <label className="text-sm font-semibold">Nhóm độ tuổi</label>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {[
            { value: "child", label: "Trẻ em", desc: "Dưới 18 tuổi" },
            { value: "adult", label: "Người trưởng thành", desc: "Từ 18 đến 60 tuổi" },
            { value: "elderly", label: "Người cao tuổi", desc: "Trên 60 tuổi" },
          ].map((item) => {
            const isSelected = ageGroup === item.value;
            return (
              <button
                type="button"
                key={item.value}
                onClick={() => setAgeGroup(item.value as AgeGroup)}
                className={`flex flex-col text-left p-3.5 rounded-xl border transition-all ${
                  isSelected
                    ? "border-primary bg-primary/5 ring-2 ring-primary/20 text-foreground"
                    : "border-border hover:bg-muted/50 text-muted-foreground hover:text-foreground"
                }`}
              >
                <div className="flex items-center justify-between w-full">
                  <span className="font-medium text-sm text-foreground">{item.label}</span>
                  {isSelected && <CheckCircle2 className="w-4 h-4 text-primary" />}
                </div>
                <span className="text-xs mt-1 text-muted-foreground">{item.desc}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* 3. Sensitivity Level (Theo quyết định CHANGELOG 2026-09-10) */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <label className="text-sm font-semibold">Mức độ nhạy cảm với không khí</label>
          <span className="text-xs text-muted-foreground">Điều chỉnh ngưỡng cảnh báo sớm hơn</span>
        </div>
        <div className="grid grid-cols-2 gap-3">
          {[
            { value: "normal", label: "Bình thường", desc: "Phản ứng thông thường theo mức khuyến cáo chung" },
            { value: "high", label: "Nhạy cảm hơn", desc: "Dễ kích ứng, ho, ngứa mắt khi bụi mịn tăng" },
          ].map((item) => {
            const isSelected = sensitivity === item.value;
            return (
              <button
                type="button"
                key={item.value}
                onClick={() => setSensitivity(item.value as SensitivityLevel)}
                className={`flex flex-col text-left p-3.5 rounded-xl border transition-all ${
                  isSelected
                    ? "border-primary bg-primary/5 ring-2 ring-primary/20 text-foreground"
                    : "border-border hover:bg-muted/50 text-muted-foreground hover:text-foreground"
                }`}
              >
                <div className="flex items-center justify-between w-full">
                  <span className="font-medium text-sm text-foreground">{item.label}</span>
                  {isSelected && <CheckCircle2 className="w-4 h-4 text-primary" />}
                </div>
                <span className="text-xs mt-1 text-muted-foreground">{item.desc}</span>
              </button>
            );
          })}
        </div>
      </div>

      <Button type="submit" disabled={isSubmitting} className="w-full h-11 text-base">
        {isSubmitting ? "Đang lưu..." : submitButtonText}
      </Button>
    </form>
  );
}
