import { c as debug, s as error } from "./context-C4spomkL.js";
import { r as spinQueueResetMutex, t as resetQueues } from "./reset-p1J0pPAX.js";

//#region src/adapters/lib/patch-history.ts
const historyUpdateMarker = "__nuqs__";
function getSearchParams(url) {
	if (url instanceof URL) return url.searchParams;
	if (url.startsWith("?")) return new URLSearchParams(url);
	try {
		return new URL(url, location.origin).searchParams;
	} catch {
		return new URLSearchParams(url);
	}
}
function shouldPatchHistory(adapter) {
	if (typeof history === "undefined") return false;
	if (history.nuqs?.version && history.nuqs.version !== "2.8.9") {
		console.error(error(409), history.nuqs.version, `2.8.9`, adapter);
		return false;
	}
	if (history.nuqs?.adapters?.includes(adapter)) return false;
	return true;
}
function markHistoryAsPatched(adapter) {
	history.nuqs = history.nuqs ?? {
		version: "2.8.9",
		adapters: []
	};
	history.nuqs.adapters.push(adapter);
}
function patchHistory(emitter, adapter) {
	if (!shouldPatchHistory(adapter)) return;
	let lastSearchSeen = typeof location === "object" ? location.search : "";
	emitter.on("update", (search) => {
		const searchString = search.toString();
		lastSearchSeen = searchString.length ? "?" + searchString : "";
	});
	window.addEventListener("popstate", () => {
		lastSearchSeen = location.search;
		resetQueues();
	});
	debug("[nuqs %s] Patching history (%s adapter)", "2.8.9", adapter);
	function sync(url) {
		spinQueueResetMutex();
		try {
			if (new URL(url, location.origin).search === lastSearchSeen) return;
		} catch {}
		try {
			emitter.emit("update", getSearchParams(url));
		} catch (e) {
			console.error(e);
		}
	}
	const originalPushState = history.pushState;
	const originalReplaceState = history.replaceState;
	history.pushState = function nuqs_pushState(state, marker, url) {
		originalPushState.call(history, state, "", url);
		if (url && marker !== historyUpdateMarker) sync(url);
	};
	history.replaceState = function nuqs_replaceState(state, marker, url) {
		originalReplaceState.call(history, state, "", url);
		if (url && marker !== historyUpdateMarker) sync(url);
	};
	markHistoryAsPatched(adapter);
}

//#endregion
export { shouldPatchHistory as i, markHistoryAsPatched as n, patchHistory as r, historyUpdateMarker as t };
//# sourceMappingURL=patch-history-CLy38Aj7.js.map