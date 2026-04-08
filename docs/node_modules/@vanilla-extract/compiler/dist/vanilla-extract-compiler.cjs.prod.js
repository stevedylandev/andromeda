'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var assert = require('assert');
var path = require('path');
var transformCss = require('@vanilla-extract/css/transformCss');
var integration = require('@vanilla-extract/integration');

function _interopDefault (e) { return e && e.__esModule ? e : { 'default': e }; }

var assert__default = /*#__PURE__*/_interopDefault(assert);

const queue = [];
let isProcessingQueue = false;
async function lock(fn) {
  return new Promise((resolve, reject) => {
    const queueFn = async () => {
      try {
        const result = await fn();
        resolve(result);
      } catch (error) {
        reject(error);
      } finally {
        isProcessingQueue = false;
        processQueue();
      }
    };
    queue.push(queueFn);
    if (!isProcessingQueue) {
      processQueue();
    }
  });
}
async function processQueue() {
  if (isProcessingQueue || queue.length === 0) {
    return;
  }
  isProcessingQueue = true;
  const fn = queue.shift();
  await fn();
}

const globalAdapterIdentifier = '__vanilla_globalCssAdapter__';
const createModuleScanner = () => {
  const cache = new Map();
  const scanModule = (moduleNode, path = []) => {
    var _moduleNode$id;
    const watchFiles = new Set();
    const cacheKey = moduleNode.id ?? moduleNode.file;
    if (!cacheKey || (_moduleNode$id = moduleNode.id) !== null && _moduleNode$id !== void 0 && _moduleNode$id.includes('@vanilla-extract/') || path.includes(cacheKey)) {
      return {
        cssDeps: [],
        watchFiles
      };
    }
    if (cache.has(cacheKey)) {
      return cache.get(cacheKey);
    }
    cache.set(cacheKey, {
      cssDeps: [],
      watchFiles
    });
    const cssDeps = new Set();
    const currentPath = [...path, cacheKey];
    for (const dependencyNode of moduleNode.importedModules) {
      const {
        cssDeps: dependencyCssDeps,
        watchFiles: dependencyWatchFiles
      } = scanModule(dependencyNode, currentPath);
      dependencyCssDeps.forEach(file => cssDeps.add(file));
      dependencyWatchFiles.forEach(file => watchFiles.add(file));
    }
    const cssDepsArray = Array.from(cssDeps);
    if (moduleNode.id && integration.cssFileFilter.test(moduleNode.id)) {
      cssDepsArray.push(moduleNode.id);
    }
    if (moduleNode.file) {
      watchFiles.add(moduleNode.file);
    }
    const scanResult = {
      cssDeps: cssDepsArray,
      watchFiles
    };
    cache.set(cacheKey, scanResult);
    return scanResult;
  };
  return scanModule;
};
const createViteServer = async ({
  root,
  identifiers,
  viteConfig,
  enableFileWatcher = true
}) => {
  var _viteConfig$server, _viteConfig$server2, _viteConfig$build;
  const pkg = integration.getPackageInfo(root);
  const vite = await import('vite');
  const server = await vite.createServer({
    ...viteConfig,
    // The vite-node server should not rewrite imported asset URLs within VE stylesheets.
    // Doing so interferes with Vite's resolution and bundling of these assets at build time.
    base: undefined,
    configFile: false,
    root,
    // Don't include HTML middlewares
    appType: 'custom',
    server: {
      middlewareMode: (_viteConfig$server = viteConfig.server) === null || _viteConfig$server === void 0 ? void 0 : _viteConfig$server.middlewareMode,
      hmr: false,
      watch: enableFileWatcher ? (_viteConfig$server2 = viteConfig.server) === null || _viteConfig$server2 === void 0 ? void 0 : _viteConfig$server2.watch : null
    },
    logLevel: 'silent',
    optimizeDeps: {
      noDiscovery: true
    },
    build: {
      dynamicImportVarsOptions: {
        // Temporary workaround for https://github.com/vitejs/vite/issues/19245.
        // Shouldn't affect functionality as it's equivalent to the default value.
        // Can be removed once https://github.com/vitejs/vite/pull/19247 is released.
        exclude: [/node_modules/]
      },
      assetsInlineLimit: (_viteConfig$build = viteConfig.build) === null || _viteConfig$build === void 0 ? void 0 : _viteConfig$build.assetsInlineLimit
    },
    ssr: {
      noExternal: true
    },
    plugins: [{
      name: 'vanilla-extract-externalize',
      enforce: 'pre',
      async resolveId(source, importer) {
        if (source.startsWith('@vanilla-extract/')) {
          const result = await this.resolve(source, importer, {
            skipSelf: true
          });
          return result ? {
            ...result,
            external: true
          } : null;
        }
      }
    }, {
      name: 'vanilla-extract-transform',
      async transform(code, id) {
        if (integration.cssFileFilter.test(id)) {
          const filescopedCode = await integration.transform({
            source: code,
            rootPath: root,
            filePath: id,
            packageName: pkg.name,
            identOption: identifiers,
            globalAdapterIdentifier
          });
          return filescopedCode;
        }
      }
    }, ...(viteConfig.plugins ?? [])]
  });

  // this is need to initialize the plugins
  await server.pluginContainer.buildStart({});
  const {
    ViteNodeRunner
  } = await import('vite-node/client');
  const {
    ViteNodeServer
  } = await import('vite-node/server');
  const node = new ViteNodeServer(server);
  class ViteNodeRunnerWithContext extends ViteNodeRunner {
    prepareContext(context) {
      return {
        ...super.prepareContext(context),
        [globalAdapterIdentifier]: this.cssAdapter
      };
    }
  }
  const runner = new ViteNodeRunnerWithContext({
    root,
    base: server.config.base,
    fetchModule(id) {
      return node.fetchModule(id);
    },
    resolveId(id, importer) {
      return node.resolveId(id, importer);
    }
  });
  if (enableFileWatcher) {
    server.watcher.on('change', filePath => {
      runner.moduleCache.invalidateDepTree([filePath]);
    });
  }
  return {
    server,
    runner
  };
};
class NormalizedMap extends Map {
  constructor(root) {
    super();
    this.root = root;
  }
  #normalizePath(filePath) {
    return integration.normalizePath(path.isAbsolute(filePath) ? filePath : path.join(this.root, filePath));
  }
  get(filePath) {
    filePath = this.#normalizePath(filePath);
    return super.get(filePath);
  }
  set(filePath, value) {
    filePath = this.#normalizePath(filePath);
    return super.set(filePath, value);
  }
}
const createCompiler = ({
  root,
  identifiers = 'debug',
  cssImportSpecifier = filePath => filePath + '.vanilla.css',
  unstable_splitCssPerRule: splitCssPerRule = false,
  viteConfig,
  enableFileWatcher,
  viteResolve,
  vitePlugins
}) => {
  assert__default["default"](!(viteConfig && (viteResolve || vitePlugins)), 'viteConfig cannot be used with viteResolve or vitePlugins');
  const vitePromise = createViteServer({
    root,
    identifiers,
    viteConfig: viteConfig ?? {
      resolve: viteResolve,
      plugins: vitePlugins
    },
    enableFileWatcher
  });
  const processVanillaFileCache = new Map();
  const cssCache = new NormalizedMap(root);
  const classRegistrationsByModuleId = new NormalizedMap(root);
  return {
    async processVanillaFile(filePath, options = {}) {
      const {
        server,
        runner
      } = await vitePromise;
      filePath = path.isAbsolute(filePath) ? filePath : path.join(root, filePath);
      const outputCss = options.outputCss ?? true;
      const cacheKey = Object.entries({
        filePath,
        outputCss
      }).map(entry => entry.join('=')).join('|');
      const cachedFile = processVanillaFileCache.get(cacheKey);
      if (cachedFile) {
        const moduleNode = server.moduleGraph.getModuleById(filePath);
        if (cachedFile.lastInvalidationTimestamp === (moduleNode === null || moduleNode === void 0 ? void 0 : moduleNode.lastInvalidationTimestamp)) {
          return cachedFile.result;
        }
      }
      const cssByModuleId = new NormalizedMap(root);
      const localClassNames = new Set();
      const composedClassLists = [];
      const cssAdapter = {
        getIdentOption: () => identifiers,
        onBeginFileScope: fileScope => {
          // Before evaluating a file, reset the cache for it
          const moduleId = integration.normalizePath(fileScope.filePath);
          cssByModuleId.set(moduleId, []);
          classRegistrationsByModuleId.set(moduleId, {
            localClassNames: new Set(),
            composedClassLists: []
          });
        },
        onEndFileScope: fileScope => {
          // For backwards compatibility, ensure the cache is populated even if
          // a file didn't contain any CSS. This is to ensure that the only
          // error messages shown in older versions are the ones below.
          const moduleId = integration.normalizePath(fileScope.filePath);
          const cssObjs = cssByModuleId.get(moduleId) ?? [];
          cssByModuleId.set(moduleId, cssObjs);
        },
        registerClassName: (className, fileScope) => {
          if (!fileScope) {
            throw new Error('Your version of @vanilla-extract/css must be at least v1.10.0. Please update to a compatible version.');
          }
          localClassNames.add(className);
          const moduleId = integration.normalizePath(fileScope.filePath);
          classRegistrationsByModuleId.get(moduleId).localClassNames.add(className);
        },
        registerComposition: (composedClassList, fileScope) => {
          if (!fileScope) {
            throw new Error('Your version of @vanilla-extract/css must be at least v1.10.0. Please update to a compatible version.');
          }
          composedClassLists.push(composedClassList);
          const moduleId = integration.normalizePath(fileScope.filePath);
          classRegistrationsByModuleId.get(moduleId).composedClassLists.push(composedClassList);
        },
        markCompositionUsed: () => {
          // This compiler currently retains all composition classes
        },
        appendCss: (css, fileScope) => {
          const moduleId = integration.normalizePath(fileScope.filePath);
          const cssObjs = cssByModuleId.get(moduleId) ?? [];
          cssObjs.push(css);
          cssByModuleId.set(moduleId, cssObjs);
        }
      };
      const {
        fileExports,
        cssImports,
        watchFiles,
        lastInvalidationTimestamp
      } = await lock(async () => {
        runner.cssAdapter = cssAdapter;
        const fileExports = await runner.executeFile(filePath);
        const moduleId = integration.normalizePath(filePath);
        const moduleNode = server.moduleGraph.getModuleById(moduleId);
        if (!moduleNode) {
          throw new Error(`Can't find ModuleNode for ${filePath}`);
        }
        const cssImports = [];
        const orderedComposedClassLists = [];
        const scanModule = createModuleScanner();
        const {
          cssDeps,
          watchFiles
        } = scanModule(moduleNode);
        for (const cssDep of cssDeps) {
          const cssDepModuleId = integration.normalizePath(cssDep);
          const cssObjs = cssByModuleId.get(cssDepModuleId);
          const cachedCss = cssCache.get(cssDepModuleId);
          const cachedClassRegistrations = classRegistrationsByModuleId.get(cssDepModuleId);
          if (cachedClassRegistrations) {
            orderedComposedClassLists.push(...cachedClassRegistrations.composedClassLists);
          }
          if (!cssObjs && !cachedCss && !cachedClassRegistrations) {
            continue;
          }
          if (cssObjs) {
            let cssRules = [];
            if (cssObjs.length > 0) {
              cssRules = transformCss.transformCss({
                localClassNames: Array.from(localClassNames),
                composedClassLists: orderedComposedClassLists,
                cssObjs
              });
            }
            cssCache.set(cssDepModuleId, {
              css: cssRules.join('\n'),
              cssRules
            });
          } else if (cachedClassRegistrations) {
            cachedClassRegistrations.localClassNames.forEach(localClassName => {
              localClassNames.add(localClassName);
            });
            composedClassLists.push(...cachedClassRegistrations.composedClassLists);
          }
          const {
            css = '',
            cssRules = []
          } = cssCache.get(cssDepModuleId) ?? {};
          if (cssObjs || css) {
            if (splitCssPerRule) {
              const importSpecifiers = await Promise.all(cssRules.map((rule, i) => cssImportSpecifier(cssDepModuleId + `#${i}`, rule)));
              for (const specifier of importSpecifiers) {
                cssImports.push(`import '${specifier}';`);
              }
            } else {
              const specifier = await cssImportSpecifier(cssDepModuleId, css);
              cssImports.push(`import '${specifier}';`);
            }
          }
        }
        return {
          fileExports,
          cssImports: outputCss ? cssImports : [],
          watchFiles,
          lastInvalidationTimestamp: moduleNode.lastInvalidationTimestamp
        };
      });
      const result = {
        source: integration.serializeVanillaModule(cssImports, fileExports, null // This compiler currently retains all composition classes
        ),
        watchFiles
      };
      processVanillaFileCache.set(cacheKey, {
        lastInvalidationTimestamp,
        result
      });
      return result;
    },
    async unstable_invalidateAllModules() {
      const {
        server,
        runner
      } = await vitePromise;
      for (const [key] of runner.moduleCache.entries()) {
        if (!key.includes('node_modules')) {
          runner.moduleCache.delete(key);
        }
      }
      for (const [id, moduleNode] of server.moduleGraph.idToModuleMap) {
        if (!id.includes('node_modules')) {
          server.moduleGraph.invalidateModule(moduleNode);
        }
      }
    },
    getCssForFile(filePath) {
      filePath = path.isAbsolute(filePath) ? filePath : path.join(root, filePath);
      const moduleId = integration.normalizePath(filePath);
      const result = cssCache.get(moduleId);
      if (!result) {
        throw new Error(`No CSS for file: ${filePath}`);
      }
      return {
        css: result.css,
        filePath: filePath,
        resolveDir: root
      };
    },
    async close() {
      const {
        server
      } = await vitePromise;
      await server.close();
    },
    getAllCss() {
      let allCss = '';
      for (const {
        css
      } of cssCache.values()) {
        if (css) {
          allCss += css + '\n';
        }
      }
      return allCss;
    },
    /**
     * Returns an importer tree based off the compiler's module graph. We can't use the
     * consuming Vite dev server's module graph as it ends up modified by the `transform` hook to a
     * point where we can't reconstruct the original importer chain.
     */
    async findImporterTree(filePath, transformedModules) {
      const {
        server
      } = await vitePromise;

      // The compiler's module graph is always a subset of the consuming Vite dev server's module
      // graph, so this early exit will be hit for any modules that aren't involved in compiling VE
      // modules
      const moduleNode = server.moduleGraph.getModuleById(integration.normalizePath(filePath));
      if (!moduleNode) {
        return new Set();
      }
      return _findImporterTree(moduleNode, transformedModules);
    }
  };
};
function _findImporterTree(moduleNode, transformedModules, visited = new Set()) {
  const result = new Set();
  if (!moduleNode.id || visited.has(moduleNode.id)) {
    return result;
  }

  // Include the starting module in the tree
  result.add(moduleNode);
  visited.add(moduleNode.id);

  // Stop if we hit a transformed module as this is a VE module boundary that we don't
  // need to invalidate past
  if (transformedModules.has(moduleNode.id)) {
    return result;
  }
  for (const importer of moduleNode.importers) {
    const chain = _findImporterTree(importer, transformedModules, visited);
    for (const mod of chain) {
      result.add(mod);
    }
  }
  return result;
}

exports.createCompiler = createCompiler;
