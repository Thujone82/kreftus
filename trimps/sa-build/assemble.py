import time
from pathlib import Path

here = Path(__file__).resolve().parent
trimps = here.parent

shell_path = here / "shell.html"
lz_path = trimps / "lz-string.js"
app_path = here / "app.js"
data_path = here / "data.json"

shell = shell_path.read_text(encoding="utf-8")
lz = lz_path.read_text(encoding="utf-8")
app = app_path.read_text(encoding="utf-8")
data = data_path.read_text(encoding="utf-8")

for token in ("/*__LZ__*/", "/*__DATA__*/", "/*__APP__*/"):
    if token not in shell:
        raise RuntimeError(f"Missing placeholder token {token} in shell.html")

html = (
    shell.replace("/*__LZ__*/", lz)
    .replace("/*__DATA__*/", data)
    .replace("/*__APP__*/", app)
)
html = html.replace("\r\n", "\n").replace("\r", "\n").replace("\n", "\r\n")

out = trimps / "sa.html"
raw = html.encode("utf-8")
for attempt in range(3):
    try:
        out.write_bytes(raw)
        break
    except OSError:
        if attempt == 2:
            raise
        time.sleep(0.2)
lines = html.count("\r\n") + 1
print(
    f"Inputs   : shell={shell_path.stat().st_size:,}B | app={app_path.stat().st_size:,}B | "
    f"data={data_path.stat().st_size:,}B | lz={lz_path.stat().st_size:,}B"
)
print(f"Assemble : Wrote sa.html ({len(raw):,} bytes, {lines:,} lines, CRLF)")