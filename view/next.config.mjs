if (!process.env.HOST) {
  process.env.HOST = '0.0.0.0';
  console.log(`[next.config] Using default HOST=${process.env.HOST}`);
}

/** @type {import('next').NextConfig} */
const nextConfig = {
  // can add further config here
};

export default nextConfig;
