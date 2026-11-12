// Shared andromeda service worker.
//
// Strategy:
//   - HTML navigations: network-first. Successful responses are cached, so
//     any page visited while online keeps working offline. Uncached pages
//     fall back to /offline.
//   - Static assets (css/js/fonts/images): stale-while-revalidate.
//   - Auth surfaces (/admin, /login, /logout), APIs, non-GET requests, and
//     cross-origin requests are never intercepted or cached.
//
// Bump CACHE_VERSION to invalidate all caches when this file changes.

const CACHE_VERSION = 'v1';
const SHELL_CACHE = 'andromeda-shell-' + CACHE_VERSION;
const PAGES_CACHE = 'andromeda-pages-' + CACHE_VERSION;
const ASSETS_CACHE = 'andromeda-assets-' + CACHE_VERSION;
const MAX_PAGE_ENTRIES = 60;

const OFFLINE_URL = '/offline';

// App-shell assets attempted at install. Entries missing in a given app are
// skipped, so the same worker works across apps with different layouts.
const SHELL_ASSETS = [
  OFFLINE_URL,
  '/static/site.webmanifest',
  '/static/styles.css',
  '/assets/darkmatter.css',
  '/assets/kepler.css',
];

// Requests the worker never touches.
const SKIP_PATTERNS = [
  /^\/admin(\/|$)/,
  /^\/login$/,
  /^\/logout$/,
  /^\/api\//,
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(SHELL_CACHE);
      await Promise.allSettled(
        SHELL_ASSETS.map(async (url) => {
          const res = await fetch(url, { cache: 'reload' });
          if (res.ok) await cache.put(url, res);
        })
      );
      await self.skipWaiting();
    })()
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      const keep = [SHELL_CACHE, PAGES_CACHE, ASSETS_CACHE];
      const names = await caches.keys();
      await Promise.all(
        names.filter((n) => !keep.includes(n)).map((n) => caches.delete(n))
      );
      await self.clients.claim();
    })()
  );
});

self.addEventListener('fetch', (event) => {
  if (!shouldHandle(event.request)) return;
  if (event.request.mode === 'navigate') {
    event.respondWith(handleNavigation(event.request));
  } else {
    event.respondWith(staleWhileRevalidate(event.request));
  }
});

function shouldHandle(request) {
  if (request.method !== 'GET') return false;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return false;
  return !SKIP_PATTERNS.some((re) => re.test(url.pathname));
}

// Network-first for pages: fresh when online, cached copy when offline.
async function handleNavigation(request) {
  const cache = await caches.open(PAGES_CACHE);
  try {
    const response = await fetch(request);
    if (response.ok) {
      cache.put(request, response.clone());
      trimCache(cache, MAX_PAGE_ENTRIES);
    }
    return response;
  } catch (err) {
    const cached = await cache.match(request);
    if (cached) return cached;
    const shell = await caches.open(SHELL_CACHE);
    const offline = await shell.match(OFFLINE_URL);
    return offline || Response.error();
  }
}

// Stale-while-revalidate for static assets.
async function staleWhileRevalidate(request) {
  const cache = await caches.open(ASSETS_CACHE);
  const cached = await cache.match(request);
  const network = fetch(request)
    .then((response) => {
      if (response.ok && isCacheableAsset(response)) {
        cache.put(request, response.clone());
      }
      return response;
    })
    .catch(() => null);
  if (cached) return cached;
  const response = await network;
  return response || new Response('offline', { status: 503, statusText: 'Offline' });
}

function isCacheableAsset(response) {
  const type = (response.headers.get('content-type') || '').toLowerCase();
  return (
    type.startsWith('text/css') ||
    type.startsWith('image/') ||
    type.startsWith('font/') ||
    type.startsWith('application/javascript') ||
    type.startsWith('text/javascript')
  );
}

// Keep the pages cache bounded by evicting the oldest entries.
async function trimCache(cache, max) {
  const keys = await cache.keys();
  if (keys.length <= max) return;
  for (const key of keys.slice(0, keys.length - max)) {
    await cache.delete(key);
  }
}
