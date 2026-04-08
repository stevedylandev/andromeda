import { c as debug } from "./context-C4spomkL.js";
import { t as compareQuery } from "./compare-Br3z3FUS.js";

//#region src/adapters/lib/key-isolation.ts
function applyChange(newValue, keys, copy) {
	return (oldValue) => {
		if (!(keys.length === 0 ? true : keys.some((key) => !compareQuery(oldValue.getAll(key), newValue.getAll(key))))) {
			debug("[nuqs `%s`] no change, returning previous", keys.join(","), oldValue);
			return oldValue;
		}
		const filtered = filterSearchParams(newValue, keys, copy);
		debug(`[nuqs \`%s\`] subbed search params change
  from %O
  to   %O`, keys.join(","), oldValue, filtered);
		return filtered;
	};
}
function filterSearchParams(search, keys, copy) {
	if (keys.length === 0) return search;
	const filtered = copy ? new URLSearchParams(search) : search;
	for (const key of search.keys()) if (!keys.includes(key)) filtered.delete(key);
	return filtered;
}

//#endregion
export { filterSearchParams as n, applyChange as t };
//# sourceMappingURL=key-isolation-Y_bk4Yva.js.map