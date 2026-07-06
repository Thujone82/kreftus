// Service worker registration + "Update available" banner.
//
// Flow:
//   1. Register /heritage/service-worker.js on load.
//   2. On 'updatefound', watch for the new worker to reach 'installed' while
//      an active controller exists -> show the update banner.
//   3. The Reload button asks the waiting worker to skipWaiting, then reloads
//      once the new worker takes control. This avoids the "two reloads needed"
//      issue with stale-while-revalidate app shells.
//
// IndexedDB (the user's found/notes) is never touched by the service worker,
// so updates are safe.

(function (global) {
    'use strict';

    function show() {
        const el = document.getElementById('updateNotification');
        if (el) el.classList.remove('hidden');
    }
    function hide() {
        const el = document.getElementById('updateNotification');
        if (el) el.classList.add('hidden');
    }

    async function reloadToNewVersion() {
        if (!('serviceWorker' in navigator)) {
            window.location.reload();
            return;
        }
        const reg = await navigator.serviceWorker.getRegistration();
        if (!reg) { window.location.reload(); return; }

        const onController = () => {
            navigator.serviceWorker.removeEventListener('controllerchange', onController);
            window.location.reload();
        };
        navigator.serviceWorker.addEventListener('controllerchange', onController);
        setTimeout(() => {
            navigator.serviceWorker.removeEventListener('controllerchange', onController);
            window.location.reload();
        }, 3000);

        if (typeof reg.update === 'function') {
            try { await reg.update(); } catch (e) { /* ignore */ }
        }
        if (reg.waiting) {
            reg.waiting.postMessage({ type: 'SKIP_WAITING' });
        } else if (reg.installing) {
            reg.installing.addEventListener('statechange', () => {
                if (reg.waiting) reg.waiting.postMessage({ type: 'SKIP_WAITING' });
            });
        } else {
            window.location.reload();
        }
    }

    function askForVersion() {
        return new Promise((resolve) => {
            if (!('serviceWorker' in navigator) || !navigator.serviceWorker.controller) {
                return resolve(null);
            }
            const onMsg = (ev) => {
                if (ev.data && ev.data.type === 'VERSION') {
                    navigator.serviceWorker.removeEventListener('message', onMsg);
                    resolve(ev.data.version || null);
                }
            };
            navigator.serviceWorker.addEventListener('message', onMsg);
            navigator.serviceWorker.controller.postMessage({ type: 'GET_VERSION' });
            setTimeout(() => {
                navigator.serviceWorker.removeEventListener('message', onMsg);
                resolve(null);
            }, 2500);
        });
    }

    function requestUpdateCheck() {
        if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
            navigator.serviceWorker.controller.postMessage({ type: 'CHECK_UPDATE' });
        }
    }

    let registrationPromise = null;

    function register() {
        if (!('serviceWorker' in navigator)) return;

        const swPath = new URL('service-worker.js', window.location.href).pathname;

        const doRegister = () => {
            if (registrationPromise) return registrationPromise;
            registrationPromise = navigator.serviceWorker.register(swPath).then((reg) => {
                reg.addEventListener('updatefound', () => {
                    const nw = reg.installing;
                    if (!nw) return;
                    nw.addEventListener('statechange', () => {
                        if (nw.state === 'installed' && navigator.serviceWorker.controller) {
                            show();
                        }
                    });
                });
                setInterval(requestUpdateCheck, 5 * 60 * 1000);
                return reg;
            }).catch((err) => {
                registrationPromise = null;
                // Harmless when a newer register() supersedes an in-flight one.
                if (err && err.name === 'AbortError') return null;
                console.warn('Service worker registration failed:', err);
                return null;
            });
            return registrationPromise;
        };

        if (document.readyState === 'loading') {
            window.addEventListener('load', doRegister, { once: true });
        } else {
            doRegister();
        }

        navigator.serviceWorker.addEventListener('message', (ev) => {
            if (ev.data && ev.data.type === 'UPDATE_AVAILABLE') show();
        });

        const btn = document.getElementById('updateReloadBtn');
        if (btn) btn.addEventListener('click', reloadToNewVersion);
    }

    global.HeritageSW = {
        register,
        show, hide,
        reloadToNewVersion,
        askForVersion,
        requestUpdateCheck
    };
})(window);
