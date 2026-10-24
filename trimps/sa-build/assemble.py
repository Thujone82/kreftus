from pathlib import Path

here = Path(__file__).resolve().parent
trimps = here.parent
shell = (here / "shell.html").read_text(encoding="utf-8")
lz = (trimps / "lz-string.js").read_text(encoding="utf-8")
app = (here / "app.js").read_text(encoding="utf-8")
data = (here / "data.json").read_text(encoding="utf-8")
html = (
    shell.replace("/*__LZ__*/", lz)
    .replace("/*__DATA__*/", data)
    .replace("/*__APP__*/", app)
)
out = trimps / "sa.html"
out.write_text(html, encoding="utf-8")
print("wrote", out, out.stat().st_size)
