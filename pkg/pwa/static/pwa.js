// Registers the shared andromeda service worker. Included by app templates
// as <script src="/pwa.js" defer></script>.
//
// Note: service workers require a secure context (https, or localhost).
// Registration over plain http on a LAN IP will fail — it is logged to the
// console rather than thrown so production users aren't spammed.
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('/sw.js')
      .catch((err) => console.warn('service worker registration failed:', err.message));
  });
}
