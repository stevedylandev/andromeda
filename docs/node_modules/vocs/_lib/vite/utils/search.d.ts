import debug_ from 'debug';
import MiniSearch from 'minisearch';
import type { PluggableList } from 'unified';
import type { Frontmatter } from '../../app/types.js';
export declare const debug: debug_.Debugger;
export declare function buildIndex({ baseDir, cacheDir }: {
    baseDir: string;
    cacheDir?: string;
}): Promise<MiniSearch<any>>;
export declare function saveIndex(outDir: string, index: MiniSearch, { cacheDir }?: {
    cacheDir?: string;
}): any;
export declare function processMdx(filePath: string, file: string, options: {
    rehypePlugins: PluggableList;
}): Promise<{
    html: string;
    frontmatter: Frontmatter;
}>;
export declare function getDocId(baseDir: string, file: string): string;
type PageSection = {
    anchor: string;
    html: string;
    isPage: boolean;
    text: string;
    titles: string[];
};
export declare function splitPageIntoSections(html: string): PageSection[];
export {};
//# sourceMappingURL=search.d.js.map