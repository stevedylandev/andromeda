import { ModuleNode, UserConfig } from 'vite';
import { IdentifierOption } from '@vanilla-extract/integration';

interface Compiler {
    processVanillaFile(filePath: string, options?: {
        outputCss?: boolean;
    }): Promise<{
        source: string;
        watchFiles: Set<string>;
    }>;
    unstable_invalidateAllModules(): Promise<void>;
    getCssForFile(virtualCssFilePath: string): {
        filePath: string;
        css: string;
    };
    close(): Promise<void>;
    getAllCss(): string;
    findImporterTree(filePath: string, transformedVeModules: Set<string>): Promise<Set<ModuleNode>>;
}
interface CreateCompilerOptions {
    root: string;
    /**
     * By default, the compiler sets up its own file watcher. This option allows you to disable it if
     * necessary, such as during a production build.
     *
     * @default true
     */
    enableFileWatcher?: boolean;
    cssImportSpecifier?: (filePath: string, css: string) => string | Promise<string>;
    /**
     * When true, generates one CSS import per rule instead of one per file.
     * This can help bundlers like Turbopack deduplicate shared CSS more effectively.
     *
     * @default false
     */
    unstable_splitCssPerRule?: boolean;
    identifiers?: IdentifierOption;
    viteConfig?: UserConfig;
    /** @deprecated */
    viteResolve?: UserConfig['resolve'];
    /** @deprecated */
    vitePlugins?: UserConfig['plugins'];
}
declare const createCompiler: ({ root, identifiers, cssImportSpecifier, unstable_splitCssPerRule: splitCssPerRule, viteConfig, enableFileWatcher, viteResolve, vitePlugins, }: CreateCompilerOptions) => Compiler;

export { type Compiler, type CreateCompilerOptions, createCompiler };
