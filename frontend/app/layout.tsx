import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/store/auth-store";

export const metadata: Metadata = {
  title: "AQI Health Monitor",
  description: "Cảnh báo chất lượng không khí cá nhân hóa theo hồ sơ sức khỏe",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="vi" className="h-full antialiased font-sans">
      <body className="min-h-full flex flex-col bg-background text-foreground">
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
