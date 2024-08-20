import { defineConfig } from "vite";

export default defineConfig({
  build: {
    manifest: true,
    rollupOptions: {
      input: "resources/main.ts",
    },
  },
  server: {
    origin: "http://localhost:8080",
  },
});
