import { useLayoutEffect, useState } from 'react';
export function useMediaQuery(query) {
    const [matches, setMatches] = useState(() => {
        if (typeof window !== 'undefined')
            return window.matchMedia(query).matches;
        return false;
    });
    useLayoutEffect(() => {
        if (typeof window === 'undefined')
            return;
        const mediaQuery = window.matchMedia(query);
        const handleChange = () => setMatches(mediaQuery.matches);
        mediaQuery.addEventListener('change', handleChange);
        return () => mediaQuery.removeEventListener('change', handleChange);
    }, [query]);
    return matches;
}
//# sourceMappingURL=useMediaQuery.js.map