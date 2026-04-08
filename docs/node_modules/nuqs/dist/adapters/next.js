'use client';

import { n as createAdapterProvider } from "../context-C4spomkL.js";
import { n as useNuqsNextAppRouterAdapter } from "../impl.app-HXOL9k0k.js";
import { n as useNuqsNextPagesRouterAdapter, t as isPagesRouter } from "../impl.pages-CQtPEwnP.js";

//#region src/adapters/next.ts
function useNuqsNextAdapter() {
	const pagesRouterImpl = useNuqsNextPagesRouterAdapter();
	const appRouterImpl = useNuqsNextAppRouterAdapter();
	return {
		searchParams: appRouterImpl.searchParams,
		updateUrl(search, options) {
			if (isPagesRouter()) return pagesRouterImpl.updateUrl(search, options);
			else return appRouterImpl.updateUrl(search, options);
		},
		autoResetQueueOnUpdate: false
	};
}
const NuqsAdapter = createAdapterProvider(useNuqsNextAdapter);

//#endregion
export { NuqsAdapter };
//# sourceMappingURL=next.js.map