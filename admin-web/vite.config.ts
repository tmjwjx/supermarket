import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

const gateway = 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/v1': {
        target: gateway,
        changeOrigin: true,
      },
    },
  },
})
