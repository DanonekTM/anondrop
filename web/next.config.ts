import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  // Enable trailing slash for consistent routing with Nginx
  trailingSlash: true,
  // Disable image optimization since we're using static export
  images: {
    unoptimized: true,
  },
  // Ensure assets are properly referenced
  assetPrefix: process.env.NEXT_PUBLIC_BASE_PATH || "",
  basePath: process.env.NEXT_PUBLIC_BASE_PATH || "",
};

export default nextConfig;
