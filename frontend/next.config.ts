import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
  // 仅开发环境把 /api 代理到 Go；静态导出产物由 Go 同源托管，无跨域。
  ...(process.env.NODE_ENV === "development"
    ? {
        async rewrites() {
          return [{ source: "/api/:path*", destination: "http://127.0.0.1:17860/api/:path*" }];
        },
      }
    : {}),
};

export default nextConfig;
