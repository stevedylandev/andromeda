import { jsx as _jsx } from "react/jsx-runtime";
import { clsx } from 'clsx';
import { forwardRef } from 'react';
import { useLocation } from 'react-router';
import { ExternalLink } from './ExternalLink.js';
import * as styles from './Link.css.js';
import { RouterLink } from './RouterLink.js';
export const Link = forwardRef((props, ref) => {
    const { hideExternalIcon, href, variant = 'accent', ...rest } = props;
    const { pathname } = useLocation();
    // External links
    if (href?.match(/^(www|https?)/))
        return (_jsx(ExternalLink, { ref: ref, className: clsx(props.className, styles.root, variant === 'accent' && styles.accent, variant === 'styleless' && styles.styleless), hideExternalIcon: hideExternalIcon, href: href, ...rest }));
    // Internal links
    const [before, after] = (href || '').split('#');
    const to = `${before ? before : pathname}${after ? `#${after}` : ''}`;
    return (_jsx(RouterLink, { ...rest, ref: ref, className: clsx(props.className, styles.root, variant === 'accent' && styles.accent, variant === 'styleless' && styles.styleless), to: to }));
});
//# sourceMappingURL=Link.js.map