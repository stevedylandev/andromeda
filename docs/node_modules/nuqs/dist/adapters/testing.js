'use client';

import { o as renderQueryString, t as context } from "../context-C4spomkL.js";
import { t as resetQueues } from "../reset-p1J0pPAX.js";
import "./custom.js";
import { createElement, useCallback, useEffect, useRef, useState } from "react";

//#region src/adapters/testing.ts
function renderInitialSearchParams(searchParams) {
	if (!searchParams) return "";
	if (typeof searchParams === "string") return searchParams;
	if (searchParams instanceof URLSearchParams) return searchParams.toString();
	return new URLSearchParams(searchParams).toString();
}
function NuqsTestingAdapter({ resetUrlUpdateQueueOnMount = true, autoResetQueueOnUpdate = true, defaultOptions, processUrlSearchParams, rateLimitFactor = 0, hasMemory = false, onUrlUpdate, children, searchParams: initialSearchParams = "" }) {
	const renderedInitialSearchParams = renderInitialSearchParams(initialSearchParams);
	const locationSearchRef = useRef(renderedInitialSearchParams);
	if (resetUrlUpdateQueueOnMount) resetQueues();
	const [searchParams, setSearchParams] = useState(() => new URLSearchParams(locationSearchRef.current));
	useEffect(() => {
		if (!hasMemory) return;
		const synced = new URLSearchParams(initialSearchParams);
		setSearchParams(synced);
		locationSearchRef.current = synced.toString();
	}, [hasMemory, renderedInitialSearchParams]);
	const updateUrl = useCallback((search, options) => {
		const queryString = renderQueryString(search);
		const searchParams = new URLSearchParams(search);
		if (hasMemory) {
			setSearchParams(searchParams);
			locationSearchRef.current = queryString;
		}
		onUrlUpdate?.({
			searchParams,
			queryString,
			options
		});
	}, [onUrlUpdate, hasMemory]);
	const getSearchParamsSnapshot = useCallback(() => {
		return new URLSearchParams(locationSearchRef.current);
	}, [renderedInitialSearchParams]);
	const useAdapter = () => ({
		searchParams,
		updateUrl,
		getSearchParamsSnapshot,
		rateLimitFactor,
		autoResetQueueOnUpdate
	});
	return createElement(context.Provider, { value: {
		useAdapter,
		defaultOptions,
		processUrlSearchParams
	} }, children);
}
/**
* A higher order component that wraps the children with the NuqsTestingAdapter
*
* It allows creating wrappers for testing purposes by providing only the
* necessary props to the NuqsTestingAdapter.
*
* Usage:
* ```tsx
* render(<MyComponent />, {
*   wrapper: withNuqsTestingAdapter({ searchParams: '?foo=bar' })
* })
* ```
*/
function withNuqsTestingAdapter(props = {}) {
	return function NuqsTestingAdapterWrapper({ children }) {
		return createElement(NuqsTestingAdapter, props, children);
	};
}

//#endregion
export { NuqsTestingAdapter, withNuqsTestingAdapter };
//# sourceMappingURL=testing.js.map