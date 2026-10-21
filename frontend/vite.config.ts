import react from '@vitejs/plugin-react';
import path from 'node:path';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@admin': path.resolve(__dirname, 'src/apps/admin'),
      '@player': path.resolve(__dirname, 'src/apps/player'),
      '@shared': path.resolve(__dirname, 'src/shared')
    }
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8005',
        changeOrigin: true,
        ws: true
      },
      '/api-admin': {
        target: 'http://127.0.0.1:8005',
        changeOrigin: true,
        ws: true
      },
      '/static': {
        target: 'http://127.0.0.1:8005',
        changeOrigin: true
      }
    }
  }
});
