import type { Module } from '../types.js';
export type PageData = {
    content?: string;
    filePath?: string;
    frontmatter: Module['frontmatter'];
    lastUpdatedAt?: number;
    previousPath?: string;
};
export declare function usePageData(): PageData;
export declare const PageDataContext: import("react").Context<PageData | undefined>;
//# sourceMappingURL=usePageData.d.js.map