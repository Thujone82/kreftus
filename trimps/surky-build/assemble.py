import argparse
import hashlib
import json
import re
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
    parser = argparse.ArgumentParser(description="Assemble self-contained surky.html from upstream snapshot + overlays.")
    parser.add_argument(
        "--out",
        default=str(HERE.parent / "surky.html"),
        help="Output HTML path (default: ../surky.html)",
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

    # 8. Populate upstream metadata token
    html = html.replace("/*__UPSTREAM_META__*/", upstream_meta_str)

    # 9. Verify self-containment before writing output
    verify_self_contained(html)

    # 10. Normalize to CRLF and write output atomically
    html = html.replace("\r\n", "\n").replace("\r", "\n").replace("\n", "\r\n")
    out_path = Path(args.out).resolve()
    out_path.parent.mkdir(parents=True, exist_ok=True)
    raw_out = html.encode("utf-8")

    for attempt in range(3):
        try:
            out_path.write_bytes(raw_out)
            break
        except OSError:
            if attempt == 2:
                raise
            time.sleep(0.2)

    lines = html.count("\r\n") + 1
    print(
        f"Inputs   : upstream={SNAP_PATH.stat().st_size:,}B | theme={THEME_CSS_PATH.stat().st_size:,}B | "
        f"overlay={OVERLAY_JS_PATH.stat().st_size:,}B | injects={HEAD_INJECT_PATH.stat().st_size + BODY_INJECT_PATH.stat().st_size:,}B"
    )
    print(f"Assemble : Wrote {out_path.name} ({len(raw_out):,} bytes, {lines:,} lines, CRLF, self-contained)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
