import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// 开发时把 /v1 转到本机 gateway 不直连后端服务
const gateway = 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': {
        target: gateway,
        changeOrigin: true,
      },
    },
  },
})
