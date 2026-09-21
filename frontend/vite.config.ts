import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// 本地开发时通过代理访问后端，避免跨域；容器部署时由 nginx 反向代理 /api。
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: false
  }
});
