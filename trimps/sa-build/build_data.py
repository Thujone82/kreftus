"""One-off: snapshot community SA sheet + item catalog from objects.js."""
from __future__ import annotations

import csv
import io
import json
import re
import sys
import time
import urllib.request
import zipfile
from pathlib import Path
from xml.etree import ElementTree as ET

HERE = Path(__file__).resolve().parent
TRIMPS = HERE.parent
ROOT = HERE
SHEET_ID = "17Z3dwnkeAmY2La-LWreybTs4Sm7BAck79EzVF_gkZzs"
XLSX_URL = f"https://docs.google.com/spreadsheets/d/{SHEET_ID}/export?format=xlsx"
NS_URI = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
NS = {"m": NS_URI}
C_TAG = f"{{{NS_URI}}}c"
V_TAG = f"{{{NS_URI}}}v"
IS_TAG = f"{{{NS_URI}}}is"
T_TAG = f"{{{NS_URI}}}t"
ROW_TAG = f"{{{NS_URI}}}row"
SD_TAG = f"{{{NS_URI}}}sheetData"

ITEM_ORDER_DISPLAY = [
    "Menacing Mask", "Sword", "Armor", "Rusty Dagger", "Fists of Goo", "Battery Stick",
    "Pants", "Raincoat", "Putrid Pouch", "Chemistry Set", "Bad Medkit", "Comfy Boots",
    "Labcoat", "Lifegiving Gem", "Mood Bracelet", "Hungering Mold", "Recycler",
    "Shining Armor", "Shock and Awl", "Spiked Gloves", "Tame Snimp", "Lich Wraps",
    "Wired Wristguards", "Aegis", "Sword and Board", "Bilious Boots", "Bloodstained Gloves",
    "Unlucky Coin", "Eelimp in a Bottle", "Big Cleaver", "The Globulator", "Metal Suit",
    "Nozzled Goggles", "Sundering Scythe", "Sacrificial Shank", "Plague Bringer",
    "Very Large Slime", "Monkimp Paw", "Grounded Crown", "Fearsome Piercer", "Bag of Nails",
    "Blessed Protector", "The Doomspring", "Snimp-Fanged Blade", "Doppelganger Signet",
    "Wrath Crafted Hatchet", "Basket of Souls", "Goo Golem", "Omni Enhancer", "Stormbringer",
    "Box of Spores", "Nullifium Armor", "Myco Mitts", "Haunted Harpoon", "Doppelganger Diadem",
    "Gaseous Greataxe", "The Fibrillator",
]

DISPLAY_TO_KEY = {
    "Menacing Mask": "Menacing_Mask",
    "Sword": "Sword",
    "Armor": "Armor",
    "Rusty Dagger": "Rusty_Dagger",
    "Fists of Goo": "Fists_of_Goo",
    "Battery Stick": "Battery_Stick",
    "Pants": "Pants",
    "Raincoat": "Raincoat",
    "Putrid Pouch": "Putrid_Pouch",
    "Chemistry Set": "Chemistry_Set",
    "Bad Medkit": "Bad_Medkit",
    "Comfy Boots": "Comfy_Boots",
    "Labcoat": "Labcoat",
    "Lifegiving Gem": "Lifegiving_Gem",
    "Mood Bracelet": "Mood_Bracelet",
    "Hungering Mold": "Hungering_Mold",
    "Recycler": "Recycler",
    "Shining Armor": "Shining_Armor",
    "Shock and Awl": "Shock_and_Awl",
    "Spiked Gloves": "Spiked_Gloves",
    "Tame Snimp": "Tame_Snimp",
    "Lich Wraps": "Lich_Wraps",
    "Wired Wristguards": "Wired_Wristguards",
    "Aegis": "Aegis",
    "Sword and Board": "Sword_and_Board",
    "Bilious Boots": "Bilious_Boots",
    "Bloodstained Gloves": "Bloodstained_Gloves",
    "Unlucky Coin": "Unlucky_Coin",
    "Eelimp in a Bottle": "Eelimp_in_a_Bottle",
    "Big Cleaver": "Big_Cleaver",
    "The Globulator": "The_Globulator",
    "Metal Suit": "Metal_Suit",
    "Nozzled Goggles": "Nozzled_Goggles",
    "Sundering Scythe": "Sundering_Scythe",
    "Sacrificial Shank": "Sacrificial_Shank",
    "Plague Bringer": "Plague_Bringer",
    "Very Large Slime": "Very_Large_Slime",
    "Monkimp Paw": "Monkimp_Paw",
    "Grounded Crown": "Grounded_Crown",
    "Fearsome Piercer": "Fearsome_Piercer",
    "Bag of Nails": "Bag_of_Nails",
    "Blessed Protector": "Blessed_Protector",
    "The Doomspring": "The_Doomspring",
    "Snimp-Fanged Blade": "Snimp__Fanged_Blade",
    "Doppelganger Signet": "Doppelganger_Signet",
    "Wrath Crafted Hatchet": "Wrath_Crafted_Hatchet",
    "Basket of Souls": "Basket_of_Souls",
    "Goo Golem": "Goo_Golem",
    "Omni Enhancer": "Omni_Enhancer",
    "Stormbringer": "Stormbringer",
    "Box of Spores": "Box_of_Spores",
    "Nullifium Armor": "Nullifium_Armor",
    "Myco Mitts": "Myco_Mitts",
    "Haunted Harpoon": "Haunted_Harpoon",
    "Doppelganger Diadem": "Doppelganger_Diadem",
    "Gaseous Greataxe": "Gaseous_Greataxe",
    "The Fibrillator": "The_Fibrillator",
}

RING_MOD_MAP = {
    "atk": "attack",
    "attack": "attack",
    "hp": "health",
    "health": "health",
    "def": "defense",
    "defense": "defense",
    "ls": "lifesteal",
    "lifesteal": "lifesteal",
    "dust": "dustMult",
    "dust mult": "dustMult",
    "dustmult": "dustMult",
}


def col_letter_to_index(col: str) -> int:
    n = 0
    for ch in col:
        if "A" <= ch <= "Z":
            n = n * 26 + (ord(ch) - 64)
        else:
            break
    return n - 1


def parse_shared_strings(zf: zipfile.ZipFile) -> list[str]:
    data = zf.read("xl/sharedStrings.xml")
    root = ET.fromstring(data)
    out = []
    for si in root.findall("m:si", NS):
        texts = [t.text or "" for t in si.findall(".//m:t", NS)]
        out.append("".join(texts))
    return out


def sheet_rows(zf: zipfile.ZipFile, path: str, shared: list[str]) -> list[list]:
    root = ET.fromstring(zf.read(path))
    sd = root.find(SD_TAG)
    if sd is None:
        return []
    rows = []
    for row in sd:
        if row.tag != ROW_TAG:
            continue
        cells = {}
        max_c = -1
        for cell in row:
            if cell.tag != C_TAG:
                continue
            ref = cell.attrib.get("r", "")
            col = col_letter_to_index(ref)
            if col < 0:
                continue
            t = cell.attrib.get("t")
            v = None
            is_elem = None
            for child in cell:
                if child.tag == V_TAG:
                    v = child
                elif child.tag == IS_TAG:
                    is_elem = child
            val = ""
            if t == "s" and v is not None and v.text is not None:
                val = shared[int(v.text)]
            elif t == "inlineStr" and is_elem is not None:
                texts = [ch.text or "" for ch in is_elem if ch.tag == T_TAG]
                val = "".join(texts)
            elif v is not None and v.text is not None:
                raw = v.text
                if "." in raw or "e" in raw.lower():
                    try:
                        val = float(raw)
                    except ValueError:
                        val = raw
                else:
                    try:
                        val = int(raw)
                    except ValueError:
                        val = raw
            cells[col] = val
            if col > max_c:
                max_c = col
        if max_c < 0:
            rows.append([])
            continue
        line = [""] * (max_c + 1)
        for i, val in cells.items():
            line[i] = val
        rows.append(line)
    return rows


def workbook_sheets(zf: zipfile.ZipFile) -> list[tuple[str, str]]:
    wb = ET.fromstring(zf.read("xl/workbook.xml"))
    rels = ET.fromstring(zf.read("xl/_rels/workbook.xml.rels"))
    rel_map = {}
    for rel in rels:
        rel_map[rel.attrib["Id"]] = rel.attrib["Target"]
    ns_r = {"r": "http://schemas.openxmlformats.org/officeDocument/2006/relationships",
            "m": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}
    out = []
    for sh in wb.findall("m:sheets/m:sheet", ns_r):
        name = sh.attrib["name"]
        rid = sh.attrib["{http://schemas.openxmlformats.org/officeDocument/2006/relationships}id"]
        target = rel_map[rid]
        if not target.startswith("xl/"):
            target = "xl/" + target.lstrip("/")
        out.append((name, target))
    return out


def parse_num(val):
    if val is None or val == "":
        return None
    if isinstance(val, (int, float)):
        return float(val)
    s = str(val).strip().replace(",", "").replace("%", "")
    if not s:
        return None
    try:
        return float(s)
    except ValueError:
        return None


def parse_int(val):
    n = parse_num(val)
    if n is None:
        return None
    return int(round(n))


def norm_header(h) -> str:
    return re.sub(r"\s+", " ", str(h or "").strip().lower())


def parse_ring_mods(*vals):
    mods = []
    for v in vals:
        if not v:
            continue
        key = RING_MOD_MAP.get(str(v).strip().lower())
        if key and key not in mods:
            mods.append(key)
    return mods


def find_header_row(rows: list[list]) -> int:
    for i, row in enumerate(rows[:8]):
        names = [str(c or "").strip() for c in row]
        hits = sum(1 for n in names if n in DISPLAY_TO_KEY)
        if hits >= 8:
            return i
    return 1


def parse_level_sheet(rows: list[list], level: int) -> list[dict]:
    if len(rows) < 2:
        return []
    hi = find_header_row(rows)
    header = [str(c or "").strip() for c in rows[hi]]
    col = {norm_header(h): i for i, h in enumerate(header) if h}
    item_cols = {}
    for i, name in enumerate(header):
        if name in DISPLAY_TO_KEY:
            item_cols[DISPLAY_TO_KEY[name]] = i

    def get(row, *names):
        for name in names:
            idx = col.get(name)
            if idx is not None and idx < len(row):
                return row[idx]
        return ""

    builds = []
    for row in rows[hi + 1 :]:
        if not row or all(c == "" or c is None for c in row):
            continue
        name = str(get(row, "author", "name") or (row[1] if len(row) > 1 else "")).strip()
        if not name and len(row) > 1:
            name = str(row[1] or "").strip()
        items = {}
        for key, idx in item_cols.items():
            if idx < len(row):
                lv = parse_int(row[idx])
                if lv and lv > 0:
                    items[key] = lv
        dust_ps = parse_num(get(row, "d/s"))
        if not items and not dust_ps:
            continue
        if name.lower() in ("", "author"):
            continue
        ring_lv = parse_int(get(row, "ring"))
        mods = parse_ring_mods(get(row, "ring1"), get(row, "ring2"), get(row, "ring3"))
        wr = parse_num(get(row, "wr"))
        if wr is not None and wr > 1:
            wr = wr / 100.0
        build = {
            "n": name,
            "i": items,
            "d": dust_ps,
            "k": parse_num(get(row, "kill time [s]", "kill time")),
            "l": parse_int(get(row, "limbs")),
            "dc": parse_num(get(row, "dust cost")),
            "sc": parse_num(get(row, "shard cost")),
            "note": str(get(row, "build notes") or "").strip(),
            "c": parse_int(get(row, "highest contract")),
        }
        if ring_lv:
            build["r"] = {"lv": ring_lv, "m": mods}
        # drop empty optional keys later
        builds.append(build)
    return builds


def extract_items_from_objects(text: str) -> dict:
    # slice items object
    start = text.find("    items: {")
    end = text.find("    bonuses: {")
    block = text[start:end]
    items = {}
    # split on top-level item keys (8 spaces)
    parts = re.split(r"\n        ([A-Za-z0-9_]+):\s*\{", block)
    # parts[0] is preamble, then name, body, name, body...
    for i in range(1, len(parts), 2):
        key = parts[i]
        body = parts[i + 1]
        # only until next sibling roughly — body includes nested braces; stop at next 8-space key is already split
        upgrade_m = re.search(r'upgrade:\s*"([^"]*)"', body)
        start_m = re.search(r"startPrice:\s*([0-9.eE+]+)", body)
        price_m = re.search(r"priceMod:\s*([0-9.eE+]+)", body)
        zone_m = re.search(r"zone:\s*(\d+)", body)
        dust_m = re.search(r'dustType:\s*"([^"]+)"', body)
        no_up = "noUpgrade:" in body
        items[key] = {
            "name": key.replace("__", "-").replace("_", " "),
            "start": float(start_m.group(1)) if start_m else 5,
            "mod": float(price_m.group(1)) if price_m else 3,
            "upgrade": upgrade_m.group(1) if upgrade_m else "",
        }
        if zone_m:
            items[key]["zone"] = int(zone_m.group(1))
        if dust_m:
            items[key]["dust"] = dust_m.group(1)
        if no_up:
            items[key]["noUp"] = True
    # display-name overrides
    for disp, key in DISPLAY_TO_KEY.items():
        if key in items:
            items[key]["name"] = disp
    return items


def compact_build(b: dict) -> dict:
    out = {"n": b["n"], "i": b["i"]}
    if b.get("d") is not None:
        out["d"] = b["d"]
    if b.get("k") is not None:
        out["k"] = b["k"]
    if b.get("l") is not None:
        out["l"] = b["l"]
    if b.get("r"):
        out["r"] = b["r"]
    if b.get("note"):
        out["note"] = b["note"]
    if b.get("c"):
        out["c"] = b["c"]
    if b.get("dc") is not None:
        out["dc"] = b["dc"]
    if b.get("sc") is not None:
        out["sc"] = b["sc"]
    return out


def main():
    fresh = "--fresh" in sys.argv
    verbose = "--verbose" in sys.argv
    cache = ROOT / "_sa.xlsx"
    if (not fresh) and cache.exists() and cache.stat().st_size > 100000:
        xlsx = cache.read_bytes()
        print(f"Workbook : Using cached _sa.xlsx ({len(xlsx):,} bytes)", flush=True)
    else:
        print("Workbook : Downloading community sheet from Google Docs...", flush=True)
        req = urllib.request.Request(XLSX_URL, headers={"User-Agent": "kreftus-sa-planner/1.0"})
        with urllib.request.urlopen(req, timeout=120) as resp:
            xlsx = resp.read()
        cache.write_bytes(xlsx)
        print(f"Workbook : Downloaded _sa.xlsx ({len(xlsx):,} bytes)", flush=True)
    zf = zipfile.ZipFile(io.BytesIO(xlsx))
    shared = parse_shared_strings(zf)
    sheets = workbook_sheets(zf)
    numeric_sheets = [(name, path) for name, path in sheets if name.isdigit()]
    total_sheets = len(numeric_sheets)
    builds = {}
    total_builds = 0
    is_tty = sys.stdout.isatty()

    for idx, (name, path) in enumerate(numeric_sheets):
        level = int(name)
        rows = sheet_rows(zf, path, shared)
        parsed = [compact_build(b) for b in parse_level_sheet(rows, level)]
        if parsed:
            builds[str(level)] = parsed
            total_builds += len(parsed)
            if verbose:
                print(f"  SA {level:>3}: {len(parsed)} builds", flush=True)

        if not verbose and ((idx + 1) % 3 == 0 or idx == total_sheets - 1 or idx == 0):
            pct = int((idx + 1) / total_sheets * 100)
            if is_tty:
                filled = pct // 5
                bar = "█" * filled + "░" * (20 - filled)
                sys.stdout.write(f"\r  Parsing  : [{bar}] {pct:>3}% (SA {level:>3}/160 · {total_builds:,} builds)")
                sys.stdout.flush()
            else:
                print(f"PROGRESS:{idx+1}:{total_sheets}:{level}:{total_builds}:{pct}", flush=True)

    if not verbose and is_tty:
        sys.stdout.write("\r" + " " * 75 + "\r")
        sys.stdout.flush()

    level_nums = sorted(int(k) for k in builds.keys())
    lvl_range = f"SA {level_nums[0]}..{level_nums[-1]}" if level_nums else "none"
    print(f"Sheets   : {total_builds:,} builds across {len(builds)} levels ({lvl_range})", flush=True)

    farm_levels = []
    index_sheets = [path for name, path in sheets if "index" in name.lower()]
    if index_sheets:
        sh_tree = ET.fromstring(zf.read(index_sheets[0]))
        for row in sh_tree.findall(".//{http://schemas.openxmlformats.org/spreadsheetml/2006/main}row"):
            for c in row.findall("{http://schemas.openxmlformats.org/spreadsheetml/2006/main}c"):
                ref = c.attrib.get("r", "")
                col = "".join([ch for ch in ref if ch.isalpha()])
                if col == "N":
                    t = c.attrib.get("t")
                    v = c.find("{http://schemas.openxmlformats.org/spreadsheetml/2006/main}v")
                    val = ""
                    if v is not None and v.text:
                        val = shared[int(v.text)] if t == "s" else v.text
                    val = str(val).strip()
                    if val and val != "Farming Levels":
                        try:
                            farm_levels.append(int(float(val)))
                        except ValueError:
                            pass
        farm_levels = sorted(list(set(farm_levels)))
    print(f"Index    : {len(farm_levels)} farming levels extracted from community Index", flush=True)

    objects = (TRIMPS / "objects.js").read_text(encoding="utf-8", errors="replace")
    items = extract_items_from_objects(objects)
    print(f"Catalog  : {len(items)} items extracted from objects.js", flush=True)
    out = {
        "sheet": f"https://docs.google.com/spreadsheets/d/{SHEET_ID}",
        "items": items,
        "builds": builds,
        "farmLevels": farm_levels,
        "itemOrder": list(DISPLAY_TO_KEY.values()),
    }
    dest = ROOT / "data.json"
    payload = json.dumps(out, separators=(",", ":"), ensure_ascii=False)
    dest.write_bytes(payload.encode("utf-8"))
    legacy = ROOT / "_sa_data.json"
    if legacy.exists():
        legacy.unlink()
    print(f"Snapshot : Wrote data.json ({dest.stat().st_size:,} bytes)", flush=True)


if __name__ == "__main__":
    main()
