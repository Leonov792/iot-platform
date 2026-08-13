import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    port: 5173,
    // проксируем api на go, чтобы в дев режиме не возиться с cors
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
})
