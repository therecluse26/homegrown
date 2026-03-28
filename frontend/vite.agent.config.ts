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
    port: 15174,
    proxy: {
      "/v1": { target: "http://localhost:15180", changeOrigin: true },
      "/hooks": { target: "http://localhost:15180", changeOrigin: true },
      "/health": { target: "http://localhost:15180", changeOrigin: true },
      "/self-service": { target: "http://localhost:4935", changeOrigin: true },
      "/sessions": { target: "http://localhost:4935", changeOrigin: true },
    },
  },
});
