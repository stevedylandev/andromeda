import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Fragment } from 'react';
import { useConfig } from '../hooks/useConfig.js';
import { primitiveColorVars, spaceVars } from '../styles/vars.css.js';
import { Icon } from './Icon.js';
import { Bluesky } from './icons/Bluesky.js';
import { Discord } from './icons/Discord.js';
import { Farcaster } from './icons/Farcaster.js';
import { GitHub } from './icons/GitHub.js';
import { Telegram } from './icons/Telegram.js';
import { Warpcast } from './icons/Warpcast.js';
import { X } from './icons/X.js';
import * as styles from './Socials.css.js';
const iconsForIcon = {
    bluesky: Bluesky,
    discord: Discord,
    farcaster: Farcaster,
    github: GitHub,
    telegram: Telegram,
    warpcast: Warpcast,
    x: X,
};
const sizesForType = {
    bluesky: '17px',
    discord: '18px',
    farcaster: '18px',
    github: '17px',
    telegram: '17px',
    warpcast: '17px',
    x: '16px',
};
export function Socials() {
    const config = useConfig();
    if (!config.socials)
        return null;
    if (config.socials.length === 0)
        return null;
    return (_jsx("div", { className: styles.root, children: config.socials.map((social, i) => (_jsxs(Fragment, { children: [i !== 0 && (_jsx("div", { style: {
                        width: '1px',
                        marginTop: spaceVars[4],
                        marginBottom: spaceVars[4],
                        backgroundColor: primitiveColorVars.border,
                    } })), _jsx("a", { className: styles.button, href: social.link, target: "_blank", rel: "noopener noreferrer", children: _jsx(Icon, { className: styles.icon, label: social.label, icon: iconsForIcon[social.icon], size: sizesForType[social.icon] || '20px' }) })] }, i))) }));
}
//# sourceMappingURL=Socials.js.map