//#region src/lib/compare.ts
function compareQuery(a, b) {
	if (a === b) return true;
	if (a === null || b === null) return false;
	if (typeof a === "string" || typeof b === "string") return false;
	if (a.length !== b.length) return false;
	return a.every((value, index) => value === b[index]);
}

//#endregion
export { compareQuery as t };
//# sourceMappingURL=compare-Br3z3FUS.js.map