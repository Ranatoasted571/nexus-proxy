import { defineConfig } from 'vite'
import { svelte, vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte({ preprocess: vitePreprocess() })],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // Inline everything into a single JS file for easy Go embedding
    rollupOptions: {
      output: {
        manualChunks: undefined,
        inlineDynamicImports: true,
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:2222',
      '/events': 'http://localhost:2222',
    }
  }
})
