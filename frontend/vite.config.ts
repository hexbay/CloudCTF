import react from '@vitejs/plugin-react';
import path from 'node:path';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined;
          if (id.includes('echarts') || id.includes('zrender')) return 'vendor-charts';
          if (id.includes('antd') || id.includes('@ant-design')) return 'vendor-antd';
          if (id.includes('react-markdown') || id.includes('remark-') || id.includes('rehype-') || id.includes('micromark')) {
            return 'vendor-markdown';
          }
          if (id.includes('xterm')) return 'vendor-terminal';
          if (id.includes('react-dom') || id.includes('react-router') || /node_modules[\\/]react[\\/]/.test(id)) {
            return 'vendor-react';
          }
          return undefined;
        }
      }
    }
  },
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
