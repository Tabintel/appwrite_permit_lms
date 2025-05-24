/** @type {import('next').NextConfig} */
const nextConfig = {
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  reactStrictMode: true,
  experimental: {
    appDir: true,
    serverActions: true,
  },
  allowedDevOrigins: ['http://localhost:3000', 'http://172.20.10.3:3000']
}

export default nextConfig
