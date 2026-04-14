import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

const host = process.env.DEV_HOST || '0.0.0.0'
const port = parseInt(process.env.DEV_PORT || '8323', 10)

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    host,
    port,
    proxy: {
      '/api': {
        target: 'http://localhost:8322',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
