import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/sub': 'http://localhost:8080',
      '/logo': 'http://localhost:8080',
    }
  },
  build: {
    outDir: 'dist',
  }
})
