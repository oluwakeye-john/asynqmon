import react from "@vitejs/plugin-react";
import type { Plugin } from "vite";
import svgr from "vite-plugin-svgr";
import { defineConfig } from "vitest/config";

const goTemplateValues = {
  __ASYNQMON_ROOT_PATH__: "/[[.RootPath]]",
  __ASYNQMON_PROMETHEUS_ADDRESS__: "/[[.PrometheusAddr]]",
  __ASYNQMON_READ_ONLY__: "/[[.ReadOnly]]",
};

const developmentValues = {
  __ASYNQMON_ROOT_PATH__: "",
  __ASYNQMON_PROMETHEUS_ADDRESS__: "",
  __ASYNQMON_READ_ONLY__: "false",
};

function serverTemplatePlugin(isBuild: boolean): Plugin {
  const values = isBuild ? goTemplateValues : developmentValues;

  return {
    name: "asynqmon-server-template",
    transformIndexHtml: {
      order: "pre",
      handler(html) {
        return Object.entries(values).reduce(
          (result, [token, value]) => result.split(token).join(value),
          html
        );
      },
    },
  };
}

export default defineConfig(({ command }) => {
  const isBuild = command === "build";

  return {
    // Production assets use relative URLs. The Go-rendered <base> element
    // supplies the runtime root path, including non-root deployments.
    base: isBuild ? "./" : "/",
    plugins: [serverTemplatePlugin(isBuild), react(), svgr()],
    build: {
      outDir: "build",
      emptyOutDir: true,
    },
    server: {
      port: 3000,
    },
    test: {
      environment: "jsdom",
      globals: true,
      setupFiles: "./src/setupTests.ts",
    },
  };
});
