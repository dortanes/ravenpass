import { readFile, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const sourceManifest = fileURLToPath(
  new URL("src/manifest.json", import.meta.url),
);

const mainWorldScript = fileURLToPath(
  new URL("src/content/webauthn-page.ts", import.meta.url),
);

const pluginName = "ravenpass:bare-main-world-script";

// Extension.js wraps content scripts in a global-setting loader and puts a script-loading bridge before each MAIN-world script.
class BareMainWorldScript {
  apply(compiler) {
    const wrapperRules = compiler.options.module.rules.filter((rule) =>
      (Array.isArray(rule?.use) ? rule.use : []).some(({ loader }) =>
        String(loader).includes("content-script-wrapper"),
      ),
    );
    if (!wrapperRules.length) {
      throw new Error(`${pluginName}: no content script wrapper rule found.`);
    }
    for (const rule of wrapperRules) {
      rule.exclude = [...(rule.exclude ?? []), mainWorldScript];
    }
    if (compiler.options.mode !== "production") return;
    compiler.hooks.afterEmit.tapPromise(pluginName, async (compilation) => {
      if (compilation.errors.length) return;
      const declared = JSON.parse(await readFile(sourceManifest, "utf8"));
      const manifestFile = join(
        compilation.outputOptions.path,
        "manifest.json",
      );
      const manifest = JSON.parse(await readFile(manifestFile, "utf8"));
      const scripts = manifest.content_scripts ?? [];
      const bridges = scripts.filter(
        (_, index) => scripts[index + 1]?.world === "MAIN",
      );
      if (scripts.length - bridges.length !== declared.content_scripts.length) {
        throw new Error(`${pluginName}: unexpected content scripts.`);
      }
      manifest.content_scripts = scripts.filter(
        (script) => !bridges.includes(script),
      );
      await writeFile(manifestFile, JSON.stringify(manifest, null, 2));
      await Promise.all(
        bridges.flatMap((bridge) =>
          bridge.js.map((file) =>
            rm(join(compilation.outputOptions.path, file)),
          ),
        ),
      );
    });
  }
}

// Chrome 121+ ignores document fields; WebKit needs the document for CryptoKey persistence.
// Extension.js strips background.scripts in its Chrome target, so restore the compiled entry.
class BackgroundEnvironments {
  apply(compiler) {
    compiler.hooks.afterEmit.tapPromise(
      "ravenpass:background-environments",
      async (compilation) => {
        if (compilation.errors.length) return;
        const manifestFile = join(
          compilation.outputOptions.path,
          "manifest.json",
        );
        const manifest = JSON.parse(await readFile(manifestFile, "utf8"));
        const worker = manifest.background?.service_worker;
        if (typeof worker !== "string" || !worker) {
          throw new Error(
            "The background service worker entry was not emitted.",
          );
        }
        await readFile(join(compilation.outputOptions.path, worker));
        manifest.background.scripts = [worker];
        manifest.background.persistent = false;
        manifest.background.preferred_environment = ["document"];
        await writeFile(manifestFile, JSON.stringify(manifest, null, 2));
      },
    );
  }
}

export default {
  // Extension.js resolves `index.ts` before a package's declared `index.js`; @scure/base ships both.
  transpilePackages: ["@ravenpass/ui", "@scure/base"],
  config: (config) => {
    config.plugins = [
      ...(config.plugins ?? []),
      new BareMainWorldScript(),
      new BackgroundEnvironments(),
    ];
    return config;
  },
};
