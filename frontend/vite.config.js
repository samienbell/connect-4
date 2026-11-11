import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In development, /api calls are forwarded to the Go backend.
// In production, nginx does the same job (see deploy/docker/nginx.conf).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': process.env.VITE_API_PROXY || 'http://localhost:8080',
    },
  },
});
