import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist/app',
  },
  server: {
    proxy: {
      '/api': `http://127.0.0.1:${process.env.KDNK_PORT ?? '8090'}`,
    },
  },
})
