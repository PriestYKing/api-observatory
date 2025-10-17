/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  reactStrictMode: true,

  // Experimental features
  experimental: {
    // Add any experimental features here if needed
  },

  // API configuration - rewrites need static destinations or use middleware instead
  // Don't use rewrites with environment variables - they're undefined at build time
  async rewrites() {
    // Only add rewrites if we're not proxying through middleware
    return [];
  },

  // Headers for WebSocket support
  async headers() {
    return [
      {
        source: "/api/:path*",
        headers: [
          { key: "Access-Control-Allow-Credentials", value: "true" },
          { key: "Access-Control-Allow-Origin", value: "*" },
          {
            key: "Access-Control-Allow-Methods",
            value: "GET,POST,PUT,DELETE,OPTIONS",
          },
          {
            key: "Access-Control-Allow-Headers",
            value: "X-Requested-With, Accept, Authorization, Content-Type",
          },
        ],
      },
    ];
  },

  // Image configuration
  images: {
    remotePatterns: [
      {
        protocol: "http",
        hostname: "localhost",
        port: "",
        pathname: "**",
      },
    ],
  },
};

module.exports = nextConfig;
