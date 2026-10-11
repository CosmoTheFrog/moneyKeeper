import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    // Туннель ngrok отдаёт запросы с чужим Host, без этого Vite их отклоняет.
    allowedHosts: ['.ngrok-free.dev'],
    proxy: { '/api': 'http://localhost:8090' },
  },
})
