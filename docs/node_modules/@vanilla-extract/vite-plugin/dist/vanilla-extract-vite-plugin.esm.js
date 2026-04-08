import path from 'path';
import { createCompiler } from '@vanilla-extract/compiler';
import { normalizePath, getPackageInfo, cssFileFilter, transform } from '@vanilla-extract/integration';

const PLUGIN_NAMESPACE = 'vite-plugin-vanilla-extract';
const virtualExtCss = '.vanilla.css';
const isVirtualId = id => id.endsWith(virtualExtCss);
const fileIdToVirtualId = id => `${id}${virtualExtCss}`;
const virtualIdToFileId = virtualId => virtualId.slice(0, -virtualExtCss.length);
const isPluginObject = plugin => typeof plugin === 'object' && plugin !== null && 'name' in plugin;
// Plugins that we know are compatible with the `vite-node` compiler
// and don't need to be filtered out.
const COMPATIBLE_PLUGINS = ['vite-tsconfig-paths'];
const defaultPluginFilter = ({
  name
}) => COMPATIBLE_PLUGINS.includes(name);
const withUserPluginFilter = ({
  mode,
  pluginFilter
}) => plugin => pluginFilter({
  name: plugin.name,
  mode
});
function vanillaExtractPlugin({
  identifiers,
  unstable_pluginFilter: pluginFilter = defaultPluginFilter,
  unstable_mode = 'emitCss'
} = {}) {
  let config;
  let configEnv;
  let server;
  let packageName;
  let compiler;
  let isBuild;
  const vitePromise = import('vite');
  const transformedModules = new Set();
  const getIdentOption = () => identifiers ?? (config.mode === 'production' ? 'short' : 'debug');
  const getAbsoluteId = filePath => {
    let resolvedId = filePath;
    if (filePath.startsWith(config.root) ||
    // In monorepos the absolute path will be outside of config.root, so we check that they have the same root on the file system
    // Paths from vite are always normalized, so we have to use the posix path separator
    path.isAbsolute(filePath) && filePath.split(path.posix.sep)[1] === config.root.split(path.posix.sep)[1]) {
      resolvedId = filePath;
    } else {
      // In SSR mode we can have paths like /app/styles.css.ts
      resolvedId = path.join(config.root, filePath);
    }
    return normalizePath(resolvedId);
  };

  /**
   * Custom invalidation function that takes a chain of importers to invalidate. If an importer is a
   * VE module, its virtual CSS is invalidated instead. Otherwise, the module is invalidated
   * normally.
   */
  const invalidateImporterChain = ({
    importerChain,
    server,
    timestamp
  }) => {
    const {
      moduleGraph
    } = server;
    const seen = new Set();
    for (const mod of importerChain) {
      if (mod.id && cssFileFilter.test(mod.id)) {
        const virtualModules = moduleGraph.getModulesByFile(fileIdToVirtualId(mod.id));
        for (const virtualModule of virtualModules ?? []) {
          moduleGraph.invalidateModule(virtualModule, seen, timestamp, true);
        }
      } else if (mod.id) {
        // `mod` is from the compiler's internal Vite server, so look up the
        // corresponding module in the consuming server's graph by ID
        const serverMod = moduleGraph.getModuleById(mod.id);
        if (serverMod) {
          moduleGraph.invalidateModule(serverMod, seen, timestamp, true);
        }
      }
    }
  };
  return [{
    name: `${PLUGIN_NAMESPACE}-inline-dev-css`,
    apply: (_, {
      command
    }) => command === 'serve' && unstable_mode === 'inlineCssInDev',
    transformIndexHtml: async () => {
      var _compiler;
      const allCss = (_compiler = compiler) === null || _compiler === void 0 ? void 0 : _compiler.getAllCss();
      if (!allCss) {
        return [];
      }
      return [{
        tag: 'style',
        children: allCss,
        attrs: {
          type: 'text/css',
          'data-vanilla-extract-inline-dev-css': true
        },
        injectTo: 'head-prepend'
      }];
    }
  }, {
    name: PLUGIN_NAMESPACE,
    configureServer(_server) {
      server = _server;
      server.watcher.on('unlink', file => {
        transformedModules.delete(normalizePath(file));
      });
    },
    config(_userConfig, _configEnv) {
      configEnv = _configEnv;
      return {
        ssr: {
          external: ['@vanilla-extract/css', '@vanilla-extract/css/fileScope', '@vanilla-extract/css/adapter']
        }
      };
    },
    configResolved(_resolvedConfig) {
      config = _resolvedConfig;
      isBuild = config.command === 'build' && !config.build.watch;
      packageName = getPackageInfo(config.root).name;
    },
    async buildStart() {
      // Ensure we re-use the compiler instance between builds, e.g. in watch mode
      if (unstable_mode !== 'transform' && !compiler) {
        var _configForViteCompile;
        const {
          loadConfigFromFile
        } = await vitePromise;
        let configForViteCompiler;

        // The user has a vite config file
        if (config.configFile) {
          const configFile = await loadConfigFromFile({
            command: config.command,
            mode: config.mode,
            isSsrBuild: configEnv.isSsrBuild
          }, config.configFile);
          configForViteCompiler = configFile === null || configFile === void 0 ? void 0 : configFile.config;
        }
        // The user is using a vite-based framework that has a custom config file
        else {
          configForViteCompiler = config.inlineConfig;
        }
        const viteConfig = {
          ...configForViteCompiler,
          plugins: (_configForViteCompile = configForViteCompiler) === null || _configForViteCompile === void 0 || (_configForViteCompile = _configForViteCompile.plugins) === null || _configForViteCompile === void 0 ? void 0 : _configForViteCompile.flat().filter(isPluginObject).filter(withUserPluginFilter({
            mode: config.mode,
            pluginFilter
          }))
        };
        compiler = createCompiler({
          root: config.root,
          identifiers: getIdentOption(),
          cssImportSpecifier: fileIdToVirtualId,
          viteConfig,
          enableFileWatcher: !isBuild
        });
      }
    },
    buildEnd() {
      // When using the rollup watcher, we don't want to close the compiler after every build.
      // Instead, we close it when the watcher is closed via the closeWatcher hook.
      if (!config.build.watch) {
        var _compiler2;
        (_compiler2 = compiler) === null || _compiler2 === void 0 || _compiler2.close();
      }
    },
    closeWatcher() {
      var _compiler3;
      return (_compiler3 = compiler) === null || _compiler3 === void 0 ? void 0 : _compiler3.close();
    },
    async transform(code, id, options) {
      const [validId] = id.split('?');
      if (!cssFileFilter.test(validId)) {
        return null;
      }
      const identOption = getIdentOption();
      const normalizedId = normalizePath(validId);
      if (unstable_mode === 'transform') {
        transformedModules.add(normalizedId);
        return transform({
          source: code,
          filePath: normalizedId,
          rootPath: config.root,
          packageName,
          identOption
        });
      }
      if (!compiler) {
        return null;
      }
      const absoluteId = getAbsoluteId(validId);
      const {
        source,
        watchFiles
      } = await compiler.processVanillaFile(absoluteId, {
        outputCss: true
      });
      transformedModules.add(normalizedId);
      const result = {
        code: source,
        map: {
          mappings: ''
        }
      };

      // We don't need to watch files or invalidate modules in build mode or during SSR
      if (isBuild || options !== null && options !== void 0 && options.ssr) {
        return result;
      }
      for (const file of watchFiles) {
        if (!file.includes('node_modules') && normalizePath(file) !== absoluteId) {
          this.addWatchFile(file);
        }
      }
      return result;
    },
    // The compiler's module graph is always a subset of the consuming Vite dev server's module
    // graph, so this early exit will be hit for any modules that aren't related to VE modules.
    async handleHotUpdate({
      file,
      server,
      timestamp
    }) {
      if (!compiler) {
        return;
      }
      const importerChain = await compiler.findImporterTree(normalizePath(file), transformedModules);
      if (importerChain.size === 0) {
        return;
      }
      invalidateImporterChain({
        importerChain,
        server,
        timestamp
      });
    },
    resolveId(source) {
      var _compiler4;
      const [validId, query] = source.split('?');
      if (!isVirtualId(validId)) return;
      const absoluteId = getAbsoluteId(validId);
      if ( // We should always have CSS for a file here.
      // The only valid scenario for a missing one is if someone had written
      // a file in their app using the .vanilla.js/.vanilla.css extension
      (_compiler4 = compiler) !== null && _compiler4 !== void 0 && _compiler4.getCssForFile(virtualIdToFileId(absoluteId))) {
        // Keep the original query string for HMR.
        return absoluteId + (query ? `?${query}` : '');
      }
    },
    load(id) {
      const [validId] = id.split('?');
      if (!isVirtualId(validId) || !compiler) return;
      const absoluteId = getAbsoluteId(validId);
      const {
        css
      } = compiler.getCssForFile(virtualIdToFileId(absoluteId));
      return css;
    }
  }];
}

export { vanillaExtractPlugin };
