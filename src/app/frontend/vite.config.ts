import { defineConfig } from "vite";
import wails from "@wailsio/runtime/plugins/vite";

// Build into ./dist, which the Go binary embeds via //go:embed all:frontend/dist.
// The wails() plugin generates/refreshes the typed Go service bindings into
// ./bindings and wires the runtime for `wails3 dev`.
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: true,
  },
  plugins: [wails("./bindings")],
});
