/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  // Server Actions are stable in Next.js v16
  serverActions: {
    bodySizeLimit: '2mb',
  },
  // Vercel will handle API routes automatically
  // No need for rewrites in production - API routes will be handled by Next.js
  async rewrites() {
    return [
      // Development: proxy API requests to local backend
      {
        source: '/api/:path*',
        destination: process.env.NODE_ENV === 'development' 
          ? 'http://localhost:8000/api/:path*' 
          : '/api/:path*',
      },
    ];
  },
  // Enable React Strict Mode
  compiler: {
    styledComponents: true,
  },
  // Headers for security
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'X-DNS-Prefetch-Control',
            value: 'on',
          },
        ],
      },
    ];
  },
  // Redirects
  async redirects() {
    return [
      {
        source: '/',
        destination: '/dashboard',
        permanent: false,
      },
    ];
  },
};

module.exports = nextConfig;
