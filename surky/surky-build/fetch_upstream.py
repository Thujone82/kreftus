import argparse
import hashlib
import json
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

HERE = Path(__file__).resolve().parent
SNAP_DIR = HERE / "upstream"
SNAP_FILE = SNAP_DIR / "index.html"
META_FILE = HERE / "upstream.json"

REPO = "nsheetz/perks"
FILE_PATH = "index.html"
USER_AGENT = "kreftus-surky-build/1.0"

REQUIRED_MARKERS = (
    "loadSave",
    "exportPerkString",
    'id="perks-col"',
    'id="aux-col"',
    'id="save-text-area"',
)


def fetch_bytes(url: str, timeout: int = 20) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read()


def fetch_commit_meta(ref: str) -> dict:
    meta = {
        "commit": ref if ref != "master" else "unknown",
        "commitDate": "Unknown",
        "commitMessage": "Unknown",
    }
    api_url = f"https://api.github.com/repos/{REPO}/commits?path={FILE_PATH}&sha={ref}&per_page=1"
    try:
        raw = fetch_bytes(api_url, timeout=15)
        arr = json.loads(raw.decode("utf-8"))
        if isinstance(arr, list) and arr:
            top = arr[0]
            meta["commit"] = top.get("sha") or meta["commit"]
            c_info = top.get("commit", {})
            c_date = (c_info.get("committer") or {}).get("date") or (c_info.get("author") or {}).get("date")
            if c_date:
                meta["commitDate"] = c_date
            msg = (c_info.get("message") or "").splitlines()
            if msg:
                meta["commitMessage"] = msg[0].strip()
    except Exception as exc:
        print(f"Upstream : Note - GitHub commit API unavailable ({exc}); falling back to raw fetch")
    return meta


def main() -> int:
    parser = argparse.ArgumentParser(description="Fetch upstream nsheetz/perks index.html snapshot.")
    parser.add_argument("--ref", default="master", help="Git ref (branch or commit SHA) to fetch (default: master)")
    args = parser.parse_args()

    meta = fetch_commit_meta(args.ref)
    target_ref = meta["commit"] if meta["commit"] != "unknown" else args.ref
    raw_url = f"https://raw.githubusercontent.com/{REPO}/{target_ref}/{FILE_PATH}"

    print(f"Upstream : Fetching {raw_url}")
    data = fetch_bytes(raw_url, timeout=25)
    text = data.decode("utf-8")

    missing = [m for m in REQUIRED_MARKERS if m not in text]
    if missing:
        raise RuntimeError(f"Upstream sanity check failed; missing markers: {', '.join(missing)}")

    # Normalize upstream snapshot to CRLF to match repo conventions
    normalized_text = text.replace("\r\n", "\n").replace("\r", "\n").replace("\n", "\r\n")
    normalized_bytes = normalized_text.encode("utf-8")
    sha256 = hashlib.sha256(normalized_bytes).hexdigest()

    SNAP_DIR.mkdir(parents=True, exist_ok=True)
    SNAP_FILE.write_bytes(normalized_bytes)

    now_iso = datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")
    meta_out = {
        "repo": REPO,
        "path": FILE_PATH,
        "commit": meta["commit"],
        "commitDate": meta["commitDate"],
        "commitMessage": meta["commitMessage"],
        "fetchedAt": now_iso,
        "sha256": sha256,
        "bytes": len(normalized_bytes),
        "lines": normalized_text.count("\r\n") + 1,
    }
    meta_json = json.dumps(meta_out, indent=2) + "\n"
    meta_json = meta_json.replace("\r\n", "\n").replace("\n", "\r\n")
    META_FILE.write_bytes(meta_json.encode("utf-8"))

    short_sha = meta["commit"][:7] if meta["commit"] != "unknown" else "unknown"
    print(f"Commit   : {REPO}@{short_sha} ({meta['commitDate']}) - {meta['commitMessage']}")
    print(f"Snapshot : Wrote upstream/index.html ({len(normalized_bytes):,} bytes, sha256 {sha256[:12]})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
