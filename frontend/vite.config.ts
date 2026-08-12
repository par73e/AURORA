import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  // 地址栏路径改为小写 /aurora/（开发地址 http://localhost:5173/aurora/；
  // 构建产物需部署在 /aurora/ 子路径下）。API 调用为绝对路径 /api，不受 base 影响。
  base: '/aurora/',
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': process.env.VITE_API_PROXY || 'http://localhost:8080',
    },
  },
})
