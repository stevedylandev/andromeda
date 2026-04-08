import { c as debug, o as renderQueryString } from "./context-C4spomkL.js";
import { t as resetQueues } from "./reset-p1J0pPAX.js";
import { useCallback, useEffect, useMemo } from "react";
import { useRouter } from "next/compat/router.js";

//#region src/adapters/next/impl.pages.ts
function isPagesRouter() {
	return typeof window.next?.router?.state?.asPath === "string";
}
let isNuqsUpdateMutex = false;
function onNavigation() {
	if (isNuqsUpdateMutex) return;
	resetQueues();
}
function useNuqsNextPagesRouterAdapter() {
	const router = useRouter();
	useEffect(() => {
		router?.events.on("routeChangeStart", onNavigation);
		router?.events.on("beforeHistoryChange", onNavigation);
		return () => {
			router?.events.off("routeChangeStart", onNavigation);
			router?.events.off("beforeHistoryChange", onNavigation);
		};
	}, []);
	return {
		searchParams: useMemo(() => {
			const searchParams = new URLSearchParams();
			if (router === null) return searchParams;
			for (const [key, value] of Object.entries(router.query)) if (typeof value === "string") searchParams.set(key, value);
			else if (Array.isArray(value)) for (const v of value) searchParams.append(key, v);
			return searchParams;
		}, [JSON.stringify(router?.query)]),
		updateUrl: useCallback((search, options) => {
			const nextRouter = window.next?.router;
			const urlParams = extractDynamicUrlParams(nextRouter.pathname, nextRouter.query);
			const asPath = getAsPathPathname(nextRouter.asPath) + renderQueryString(search) + location.hash;
			debug("[nuqs next/pages] Updating url: %s", asPath);
			const method = options.history === "push" ? nextRouter.push : nextRouter.replace;
			isNuqsUpdateMutex = true;
			method.call(nextRouter, {
				pathname: nextRouter.pathname,
				query: {
					...urlSearchParamsToObject(search),
					...urlParams
				}
			}, asPath, {
				scroll: options.scroll,
				shallow: options.shallow
			}).finally(() => {
				isNuqsUpdateMutex = false;
			});
		}, []),
		autoResetQueueOnUpdate: false
	};
}
function getAsPathPathname(asPath) {
	return asPath.replace(/#.*$/, "").replace(/\?.*$/, "");
}
function urlSearchParamsToObject(search) {
	const out = {};
	for (const key of search.keys()) {
		const values = search.getAll(key);
		if (values.length === 1) out[key] = values[0];
		else if (values.length > 1) out[key] = values;
	}
	return out;
}
/**
* Next.js pages router merges dynamic URL params with search params in its
* internal state.
* However, we need to pass just the URL params to the href part of the router
* update functions.
* This function finds the dynamic URL params placeholders in the pathname
* (eg: `/path/[foo]/[bar]`) and extracts the corresponding values from the
* query state object, leaving out any other search params.
*/
function extractDynamicUrlParams(pathname, values) {
	const paramNames = /* @__PURE__ */ new Set();
	const dynamicRegex = /\[([^\]]+)\]/g;
	const catchAllRegex = /\[\.{3}([^\]]+)\]$/;
	const optionalCatchAllRegex = /\[\[\.{3}([^\]]+)\]\]$/;
	let match;
	while ((match = dynamicRegex.exec(pathname)) !== null) {
		const paramName = match[1];
		if (paramName) paramNames.add(paramName);
	}
	const dynamicValues = Object.fromEntries(Object.entries(values).filter(([key]) => paramNames.has(key)));
	const matchCatchAll = catchAllRegex.exec(pathname);
	if (matchCatchAll && matchCatchAll[1]) {
		const key = matchCatchAll[1];
		dynamicValues[key] = values[key] ?? [];
	}
	const matchOptionalCatchAll = optionalCatchAllRegex.exec(pathname);
	if (matchOptionalCatchAll && matchOptionalCatchAll[1]) {
		const key = matchOptionalCatchAll[1];
		dynamicValues[key] = values[key] ?? [];
	}
	return dynamicValues;
}

//#endregion
export { useNuqsNextPagesRouterAdapter as n, isPagesRouter as t };
//# sourceMappingURL=impl.pages-CQtPEwnP.js.map