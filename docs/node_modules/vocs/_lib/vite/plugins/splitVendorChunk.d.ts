import type { GetManualChunk } from 'rollup';
import type { PluginOption } from 'vite';
export declare const isCSSRequest: (request: string) => boolean;
export declare class SplitVendorChunkCache {
    cache: Map<string, boolean>;
    constructor();
    reset(): void;
}
export declare function splitVendorChunk(options?: {
    cache?: SplitVendorChunkCache;
}): GetManualChunk;
export declare function splitVendorChunkPlugin(): PluginOption;
//# sourceMappingURL=splitVendorChunk.d.js.map