import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  // 地址栏路径改为 /AURORA/（开发地址 http://localhost:5173/AURORA/；
  // 构建产物需部署在 /AURORA/ 子路径下）。API 调用为绝对路径 /api，不受 base 影响。
  base: '/AURORA/',
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
