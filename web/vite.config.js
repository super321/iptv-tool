import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': 'http://localhost:8023',
      '/sub': 'http://localhost:8023',
      '/logo': 'http://localhost:8023',
    }
  },
  build: {
    outDir: 'dist',
  }
})
