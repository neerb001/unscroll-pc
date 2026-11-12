"""Zet de opdrachten (adventures), hun volgorde en de foutmeldingen van Hedy om naar één
JSON-bestand voor Unscroll. Bron: https://github.com/hedyorg/hedy (EUPL-1.2).

Gebruik: python3 maak_inhoud.py HEDY-REPO HEDY-PAKKET UITMAP"""
import ast
import json
import pathlib
import re
import sys

import yaml

repo, pakket, uit = map(pathlib.Path, sys.argv[1:4])
TALEN = ["nl", "en"]

# volgorde van de opdrachten per niveau, zoals op hedy.org
bron = (repo / "website_content.py").read_text(encoding="utf-8")
volgorde = {}
for node in ast.walk(ast.parse(bron)):
    if isinstance(node, ast.Assign) and any(getattr(t, "id", "") == "ADVENTURE_ORDER_PER_LEVEL_OLD" for t in node.targets):
        volgorde = ast.literal_eval(node.value)
volgorde = {int(k): [a for a in v if a not in ("parsons", "quiz")] for k, v in volgorde.items()}

# alles kan: tekenen (met tekens), toetsen, muziek en fouten zoeken
OVERSLAAN = set()


def woorden(taal):
    return yaml.safe_load((pakket / "data" / "keywords" / f"{taal}.yaml").read_text(encoding="utf-8"))


def vul_in(tekst, kw):
    return re.sub(r"\{(\w+)\}", lambda m: str(kw.get(m.group(1), m.group(0))), tekst)


def blokken(niveau_inhoud, kw):
    def sleutel(k):
        nr = re.search(r"_(\d+)$", k)
        return (int(nr.group(1)) if nr else 1, 0 if k.startswith("story") else 1)
    uitvoer = []
    for k in sorted(niveau_inhoud, key=sleutel):
        v = vul_in(str(niveau_inhoud[k]), kw).strip()
        if not v:
            continue
        if k.startswith("example_code"):
            uitvoer.append({"code": v.strip("`").strip()})
        elif k.startswith("story_text"):
            v = v.replace("**", "").replace("### ", "").replace("`", "")
            uitvoer.append({"tekst": v})
    return uitvoer


inhoud = {}
for taal in TALEN:
    kw = woorden(taal)
    avonturen = yaml.safe_load((repo / "content" / "adventures" / f"{taal}.yaml").read_text(encoding="utf-8"))["adventures"]
    niveaus = {}
    for niveau, lijst in sorted(volgorde.items()):
        items = []
        for naam in lijst:
            a = avonturen.get(naam) or {}
            lv = (a.get("levels") or {}).get(niveau)
            if naam in OVERSLAAN or not lv:
                continue
            b = blokken(lv, kw)
            if b:
                items.append({"sleutel": naam, "naam": vul_in(a.get("name", naam), kw), "blokken": b})
        niveaus[str(niveau)] = items
    inhoud[taal] = niveaus

# foutmeldingen uit de vertalingen van de website (de foutcodes staan in hedy/exceptions.py)
codes = set(re.findall(r"super\(\).__init__\(\s*'([^']+)'", (pakket / "exceptions.py").read_text(encoding="utf-8")))
codes |= set(re.findall(r"error_code\s*=\s*'([^']+)'", (pakket / "exceptions.py").read_text(encoding="utf-8")))


def lees_po(pad):
    vertaald, msgid, msgstr, veld = {}, None, None, None
    for regel in pad.read_text(encoding="utf-8").splitlines() + [""]:
        if regel.startswith("msgid "):
            if msgid is not None and msgstr:
                vertaald[msgid] = msgstr
            msgid, msgstr, veld = ast.literal_eval(regel[6:]), "", "id"
        elif regel.startswith("msgstr "):
            msgstr, veld = ast.literal_eval(regel[7:]), "str"
        elif regel.startswith('"') and veld:
            if veld == "id":
                msgid += ast.literal_eval(regel)
            else:
                msgstr += ast.literal_eval(regel)
        elif not regel.strip():
            if msgid is not None and msgstr:
                vertaald[msgid] = msgstr
            msgid, msgstr, veld = None, None, None
    return vertaald


fouten = {}
for taal in TALEN:
    po = lees_po(repo / "translations" / taal / "LC_MESSAGES" / "messages.po")
    fouten[taal] = {c: po[c] for c in codes if po.get(c)}

uit.mkdir(parents=True, exist_ok=True)
(uit / "inhoud.json").write_text(json.dumps(inhoud, ensure_ascii=False), encoding="utf-8")
(uit / "fouten.json").write_text(json.dumps(fouten, ensure_ascii=False), encoding="utf-8")
print("niveaus:", len(inhoud["nl"]), "opdrachten op niveau 1:", [a["sleutel"] for a in inhoud["nl"]["1"]],
      "foutmeldingen:", {t: len(f) for t, f in fouten.items()})
