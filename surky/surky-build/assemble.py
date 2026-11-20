import argparse
import hashlib
import json
import re
import shutil
import sys
import time
from pathlib import Path

HERE = Path(__file__).resolve().parent
SNAP_PATH = HERE / "upstream" / "index.html"
META_PATH = HERE / "upstream.json"
HEAD_INJECT_PATH = HERE / "head-inject.html"
BODY_INJECT_PATH = HERE / "body-inject.html"
THEME_CSS_PATH = HERE / "theme.css"
OVERLAY_JS_PATH = HERE / "overlay.js"

ANCHOR_PATTERNS = {
    "html_open": re.compile(r"<html>", re.IGNORECASE),
    "head_open": re.compile(r"<head>", re.IGNORECASE),
    "title_tag": re.compile(r"<title>.*?</title>", re.IGNORECASE | re.DOTALL),
    "style_head_close": re.compile(r"</style>\s*</head>", re.IGNORECASE),
    "body_open": re.compile(r"<body>", re.IGNORECASE),
    "main_script": re.compile(
        r'<script\s+type="text/javascript">\s*(.*?)\s*</script>',
        re.IGNORECASE | re.DOTALL,
    ),
    "body_close": re.compile(r"</body>", re.IGNORECASE),
}

CDN_SCRIPT_RE = re.compile(
    r'\s*<script\s+src="https?://[^"]+"[^>]*>\s*</script>',
    re.IGNORECASE,
)
GFONT_FACE_RE = re.compile(
    r"(?:/\*[^*]*\*/\s*)?@font-face\s*\{[^}]*fonts\.gstatic\.com[^}]*\}\s*",
    re.IGNORECASE | re.DOTALL,
)


def load_upstream_meta() -> dict:
    if not META_PATH.exists():
        return {}
    return json.loads(META_PATH.read_text(encoding="utf-8"))


def verify_anchors(raw_html: str) -> None:
    failures = []
    for name, pat in ANCHOR_PATTERNS.items():
        matches = list(pat.finditer(raw_html))
        if len(matches) != 1:
            failures.append(f"{name} (matched {len(matches)} times, expected 1)")
    if failures:
        raise RuntimeError(
            "Upstream HTML structure changed; failed anchor checks: " + ", ".join(failures)
        )


def extract_upstream_js(out_js_path: Path) -> int:
    if not SNAP_PATH.exists():
        raise FileNotFoundError(f"Missing upstream snapshot: {SNAP_PATH}")
    raw_html = SNAP_PATH.read_text(encoding="utf-8")
    verify_anchors(raw_html)
    m = ANCHOR_PATTERNS["main_script"].search(raw_html)
    assert m is not None
    js_body = m.group(1).replace("\r\n", "\n").replace("\r", "\n").replace("\n", "\r\n")
    out_js_path.write_bytes(js_body.encode("utf-8"))
    return 0


def verify_self_contained(assembled: str) -> None:
    ext_scripts = re.findall(r'<script[^>]+src=["\']https?://[^"\']+["\']', assembled, re.I)
    ext_links = re.findall(r'<link[^>]+href=["\']https?://[^"\']+["\']', assembled, re.I)
    gfonts = re.findall(r"fonts\.gstatic\.com|fonts\.googleapis\.com", assembled, re.I)
    parent_refs = re.findall(r'(?:src|href)=["\']\.\./[^"\']*["\']', assembled, re.I)
    issues = []
    if ext_scripts:
        issues.append(f"external <script src>: {ext_scripts}")
    if ext_links:
        issues.append(f"external <link href>: {ext_links}")
    if gfonts:
        issues.append("external Google Fonts reference")
    if parent_refs:
        issues.append(f"parent-relative ../ resource ref: {parent_refs}")
    if issues:
        raise RuntimeError("Self-containment check failed: " + "; ".join(issues))


def main() -> int:
    parser = argparse.ArgumentParser(description="Assemble self-contained Surky HTML (default: index.html) + PWA files from upstream snapshot + overlays.")
    parser.add_argument(
        "--out",
        default=str(HERE.parent / "index.html"),
        help="Output HTML path (default: ../index.html)",
    )
    parser.add_argument(
        "--extract-upstream-js",
        default="",
        help="If set, extract the upstream inline JS to this path and exit.",
    )
    args = parser.parse_args()

    if args.extract_upstream_js:
        return extract_upstream_js(Path(args.extract_upstream_js))

    for req_file in (SNAP_PATH, HEAD_INJECT_PATH, BODY_INJECT_PATH, THEME_CSS_PATH, OVERLAY_JS_PATH):
        if not req_file.exists():
            raise FileNotFoundError(f"Missing required build input: {req_file}")

    snap_bytes = SNAP_PATH.read_bytes()
    snap_sha = hashlib.sha256(snap_bytes).hexdigest()
    meta = load_upstream_meta()
    expected_sha = meta.get("sha256")
    if expected_sha and expected_sha != snap_sha:
        print(
            f"Warning  : upstream/index.html sha256 ({snap_sha[:8]}) differs from upstream.json ({expected_sha[:8]})"
        )

    raw_html = snap_bytes.decode("utf-8")
    verify_anchors(raw_html)
    print(f"Anchors  : Verified all {len(ANCHOR_PATTERNS)} structural injection anchors")

    head_inject = HEAD_INJECT_PATH.read_text(encoding="utf-8").strip()
    body_inject = BODY_INJECT_PATH.read_text(encoding="utf-8").strip()
    theme_css = THEME_CSS_PATH.read_text(encoding="utf-8").strip()
    overlay_js = OVERLAY_JS_PATH.read_text(encoding="utf-8").strip()

    commit_sha = (meta.get("commit") or "unknown")[:7]
    commit_date = (meta.get("commitDate") or "Unknown")[:10]
    upstream_meta_str = f"Upstream: nsheetz/perks@{commit_sha} ({commit_date})"

    # 1. Strip external Google Fonts @font-face rules and CDN <script> tags
    html, n_fonts = GFONT_FACE_RE.subn("", raw_html)
    html, n_cdn = CDN_SCRIPT_RE.subn("", html)
    print(f"Strip    : Removed {n_fonts} external @font-face rules and {n_cdn} external CDN <script> tags")

    # 2. Prepend DOCTYPE and set <html lang="en" data-theme="dark">
    banner_comment = f"<!-- Built from nsheetz/perks@{commit_sha} ({commit_date}) by surky-build/assemble.py -->"
    html = ANCHOR_PATTERNS["html_open"].sub(
        f'<!DOCTYPE html>\n{banner_comment}\n<html lang="en" data-theme="dark">',
        html,
        count=1,
    )

    # 3. Inject head metadata, CSP, inline SVG icons, and pre-paint theme loader
    html = ANCHOR_PATTERNS["head_open"].sub(
        "<head>\n" + head_inject,
        html,
        count=1,
    )

    # 4. Replace <title>
    html = ANCHOR_PATTERNS["title_tag"].sub(
        "<title>Surky Perk Optimizer — Trimps</title>",
        html,
        count=1,
    )

    # 5. Inject SA design-system theme.css right before </head>
    html = ANCHOR_PATTERNS["style_head_close"].sub(
        "</style>\n<style id=\"surky-sa-theme\">\n" + theme_css + "\n</style>\n</head>",
        html,
        count=1,
    )

    # 6. Inject sticky header, toast container, and help dialog right after <body>
    html = ANCHOR_PATTERNS["body_open"].sub(
        '<body id="top">\n' + body_inject,
        html,
        count=1,
    )

    # 7. Append overlay.js right after the main upstream script
    def _append_overlay(match: re.Match) -> str:
        return (
            match.group(0)
            + "\n<script id=\"surky-sa-overlay\">\n"
            + overlay_js
            + "\n</script>"
        )

    html = ANCHOR_PATTERNS["main_script"].sub(_append_overlay, html, count=1)

    out_path = Path(args.out).resolve()
    out_path.parent.mkdir(parents=True, exist_ok=True)
    html_name = out_path.name
    manifest_name = "surky-manifest.json"
    sw_name = "surky-sw.js"

    # 8. Prepare PWA manifest & icons and compute deterministic local-first build hash
    is_dir_index = html_name.lower() == "index.html"
    pwa_start_url = "./" if is_dir_index else f"./{html_name}"
    pwa_scope = "./" if is_dir_index else f"./{html_name}"
    pwa_id = f"/{out_path.parent.name}/" if is_dir_index else f"/{out_path.parent.name}/{html_name}"

    icon_192_src = HERE / "surky-icon-192.png"
    icon_512_src = HERE / "surky-icon-512.png"
    if out_path.parent == HERE.parent:
        icon_prefix = f"{HERE.name}/"
    else:
        icon_prefix = ""
        for icon_src in (icon_192_src, icon_512_src):
            if icon_src.exists():
                shutil.copy2(icon_src, out_path.parent / icon_src.name)

    icon_192_ref = f"{icon_prefix}surky-icon-192.png"
    icon_512_ref = f"{icon_prefix}surky-icon-512.png"

    manifest_obj = {
        "id": pwa_id,
        "name": "Surky Perk Optimizer",
        "short_name": "Surky",
        "description": "Local-first Universe 2 Radon perk calculator and optimizer for Trimps.",
        "start_url": pwa_start_url,
        "scope": pwa_scope,
        "display": "standalone",
        "background_color": "#08090c",
        "theme_color": "#08090c",
        "icons": [
            {
                "src": icon_192_ref,
                "sizes": "192x192",
                "type": "image/png",
                "purpose": "any"
            },
            {
                "src": icon_512_ref,
                "sizes": "512x512",
                "type": "image/png",
                "purpose": "any"
            }
        ]
    }
    manifest_text = json.dumps(manifest_obj, indent=2) + "\n"

    build_hasher = hashlib.sha256()
    for part in (snap_bytes, head_inject.encode("utf-8"), body_inject.encode("utf-8"), theme_css.encode("utf-8"), overlay_js.encode("utf-8"), manifest_text.encode("utf-8")):
        build_hasher.update(part)
    build_hash = build_hasher.hexdigest()[:12]

    html = html.replace("/*__UPSTREAM_META__*/", upstream_meta_str)
    html = html.replace("/*__SURKY_BUILD_HASH__*/", build_hash)
    html = html.replace("/*__SURKY_MANIFEST_HREF__*/", manifest_name)
    html = html.replace("/*__SURKY_HTML_NAME__*/", html_name)
    html = html.replace("/*__SURKY_SW_NAME__*/", sw_name)
    html = html.replace("/*__SURKY_SW_SCOPE__*/", pwa_scope)

    # 9. Verify self-containment before writing output
    verify_self_contained(html)

    # 10. Prepare local-first Cache-First Service Worker

    precache_list = [f"./{html_name}", f"./{manifest_name}", f"./{icon_192_ref}", f"./{icon_512_ref}"]
    if is_dir_index:
        precache_list.insert(0, "./")
    precache_json = json.dumps(precache_list, indent=2)

    sw_js_text = f'''/* Auto-generated by surky-build/assemble.py — Local-First Service Worker for Surky */
const BUILD_HASH = "{build_hash}";
const CACHE_NAME = "surky-pwa-" + BUILD_HASH;
const APP_SHELL_URL = "{pwa_start_url}";
const MANIFEST_URL = "{manifest_name}";
const PRECACHE_URLS = {precache_json};

self.addEventListener("install", function (event) {{
  event.waitUntil(
    caches.open(CACHE_NAME).then(function (cache) {{
      return Promise.all(
        PRECACHE_URLS.map(function (url) {{
          return cache.add(new Request(url, {{ cache: "reload" }}));
        }})
      );
    }}).then(function () {{
      // Activate immediately on first install; wait cleanly for user confirmation on upgrades
      if (!self.registration || !self.registration.active) {{
        return self.skipWaiting();
      }}
    }})
  );
}});

self.addEventListener("activate", function (event) {{
  event.waitUntil(
    caches.keys().then(function (names) {{
      return Promise.all(
        names
          .filter(function (name) {{ return name.indexOf("surky-pwa-") === 0 && name !== CACHE_NAME; }})
          .map(function (name) {{ return caches.delete(name); }})
      );
    }}).then(function () {{
      return self.clients.claim();
    }})
  );
}});

self.addEventListener("message", function (event) {{
  var data = event.data || {{}};
  if (data.type === "SKIP_WAITING") {{
    self.skipWaiting();
  }} else if (data.type === "GET_VERSION") {{
    var payload = {{ type: "VERSION", buildHash: BUILD_HASH, cacheName: CACHE_NAME }};
    if (event.ports && event.ports[0]) {{
      event.ports[0].postMessage(payload);
    }} else if (event.source) {{
      event.source.postMessage(payload);
    }}
  }}
}});

self.addEventListener("fetch", function (event) {{
  var request = event.request;
  if (request.method !== "GET") return;
  var url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  // Allow explicit update checks to bypass local cache directly to network
  if (request.cache === "no-store" || url.searchParams.has("__surky_update_check")) {{
    return;
  }}

  // Always refresh manifest when online so PWA metadata/icons/id stay current, falling back to cache offline
  if (url.pathname.endsWith("/" + MANIFEST_URL)) {{
    event.respondWith(
      caches.open(CACHE_NAME).then(function (cache) {{
        return fetch(new Request(request, {{ cache: "no-cache" }})).then(function (networkResp) {{
          if (networkResp && networkResp.ok) {{
            cache.put(request, networkResp.clone());
          }}
          return networkResp;
        }}).catch(function () {{
          return cache.match(request, {{ ignoreSearch: true }});
        }});
      }})
    );
    return;
  }}

  // Local-First (Cache-First) strategy: boot instantaneously from local CacheStorage
  event.respondWith(
    caches.open(CACHE_NAME).then(function (cache) {{
      return cache.match(request, {{ ignoreSearch: true }}).then(function (cached) {{
        if (cached) return cached;
        if (request.mode === "navigate") {{
          return cache.match(APP_SHELL_URL, {{ ignoreSearch: true }}).then(function (shellCached) {{
            if (shellCached) return shellCached;
            return fetch(request).then(function (networkResp) {{
              if (networkResp && networkResp.ok) {{
                cache.put(APP_SHELL_URL, networkResp.clone());
              }}
              return networkResp;
            }});
          }});
        }}
        return fetch(request).then(function (networkResp) {{
          if (networkResp && networkResp.ok) {{
            cache.put(request, networkResp.clone());
          }}
          return networkResp;
        }});
      }});
    }})
  );
}});
'''

    def _write_crlf(target: Path, text: str) -> bytes:
        normalized = text.replace("\r\n", "\n").replace("\r", "\n").replace("\n", "\r\n")
        raw_bytes = normalized.encode("utf-8")
        for attempt in range(3):
            try:
                target.write_bytes(raw_bytes)
                break
            except OSError:
                if attempt == 2:
                    raise
                time.sleep(0.2)
        return raw_bytes

    raw_out = _write_crlf(out_path, html)
    _write_crlf(out_path.parent / manifest_name, manifest_text)
    _write_crlf(out_path.parent / sw_name, sw_js_text)

    lines = raw_out.count(b"\r\n") + 1
    print(
        f"Inputs   : upstream={SNAP_PATH.stat().st_size:,}B | theme={THEME_CSS_PATH.stat().st_size:,}B | "
        f"overlay={OVERLAY_JS_PATH.stat().st_size:,}B | injects={HEAD_INJECT_PATH.stat().st_size + BODY_INJECT_PATH.stat().st_size:,}B"
    )
    print(f"Assemble : Wrote {out_path.name} ({len(raw_out):,} bytes, {lines:,} lines, build={build_hash}, CRLF, self-contained)")
    print(f"PWA      : Wrote {manifest_name} & {sw_name} (local-first Cache-First, scope {pwa_scope})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
