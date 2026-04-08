'use client';

import { n as createAdapterProvider } from "../../context-C4spomkL.js";
import { n as useNuqsNextAppRouterAdapter, t as NavigationSpy } from "../../impl.app-HXOL9k0k.js";
import { Suspense, createElement } from "react";

//#region src/adapters/next/app.ts
const Provider = createAdapterProvider(useNuqsNextAppRouterAdapter);
function NuqsAdapter({ children, ...adapterProps }) {
	return createElement(Provider, {
		...adapterProps,
		children: [createElement(Suspense, {
			key: "nuqs-adapter-suspense-navspy",
			children: createElement(NavigationSpy)
		}), children]
	});
}

//#endregion
export { NuqsAdapter };
//# sourceMappingURL=app.js.map