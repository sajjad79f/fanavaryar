import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// در حالت توسعه، درخواست‌های /api به بک‌اند Rust روی 8080 منتقل می‌شوند
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
