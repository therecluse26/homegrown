import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig({
  plugins: [tailwindcss(), react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5673,
    proxy: {
      "/v1": {
        target: "http://localhost:3500",
        changeOrigin: true,
      },
      "/hooks": {
        target: "http://localhost:3500",
        changeOrigin: true,
      },
      "/health": {
        target: "http://localhost:3500",
        changeOrigin: true,
      },
    },
  },
});
