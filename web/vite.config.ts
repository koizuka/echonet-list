import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'bundle',
    // Raspberry Pi向けビルド最適化
    minify: 'esbuild',              // Terserより高速
    target: 'esnext',                // 最新ブラウザ向けでトランスパイル削減
    sourcemap: false,                // プロダクションビルドでsourcemap生成をスキップ
    reportCompressedSize: false,     // gzip圧縮サイズ計算をスキップして高速化
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    host: true,
    https: {
      key: '../certs/localhost+2-key.pem',
      cert: '../certs/localhost+2.pem'
    }
  },
  define: {
    'import.meta.env.BUILD_DATE': JSON.stringify(new Date().toISOString()),
  },
})