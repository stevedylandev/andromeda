export function mergeRefs(...refs) {
    return (val) => {
        setRef(val, ...refs);
    };
}
export function setRef(val, ...refs) {
    refs.forEach((ref) => {
        if (typeof ref === 'function') {
            ref(val);
        }
        else if (ref != null) {
            ref.current = val;
        }
    });
}
//# sourceMappingURL=mergeRefs.js.map