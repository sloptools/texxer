import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  // three.js alone is above the default limit; this is a local app.
  build: { chunkSizeWarningLimit: 1500 },
  server: {
    proxy: { '/api': 'http://localhost:7777' },
  },
});
