"""Maakt hedylang geschikt voor Unscroll.

1. Python 3.11 (Debian 12). Hedy vraagt 3.12, maar gebruikt daarvan maar twee dingen: een
   'type X = ...'-regel en een backslash in een f-string.
2. Een map waarin het kind niet mag schrijven. Hedy schrijft bij elke run de samengevoegde
   grammatica naar lib/hedy/data/grammars-Total ("to ease debugging"). Op het apparaat is
   /opt/unscroll/hedy van root, dus dan stopt elk programma meteen met "Permission denied".
   Dat bestand is alleen voor Hedy's eigen ontwikkelaars; mislukt het, dan gaan we gewoon door.
"""
import pathlib
import py_compile
import re
import sys

map_ = pathlib.Path(sys.argv[1])
for f in map_.glob("*.py"):
    s = f.read_text(encoding="utf-8")
    s2 = re.sub(r"^type (\w+) = ", r"\1 = ", s, flags=re.M)
    s2 = s2.replace("ConvertToPython.indent('\\n'.join(", "ConvertToPython.indent(chr(10).join(")
    if s2 != s:
        f.write_text(s2, encoding="utf-8")
        print("aangepast:", f.name)

grammar = map_ / "grammar.py"
s = grammar.read_text(encoding="utf-8")
oud = """def write_total_grammars_file(content, *paths):
    path_ = path.join(total_grammars_dir(), *paths)
    with open(path_, "w", encoding="utf-8") as file:
        file.write(content)
"""
nieuw = """def write_total_grammars_file(content, *paths):
    # Unscroll: alleen om te debuggen; in een alleen-lezen map gewoon overslaan
    path_ = path.join(total_grammars_dir(), *paths)
    try:
        with open(path_, "w", encoding="utf-8") as file:
            file.write(content)
    except OSError:
        pass
"""
if oud not in s:
    sys.exit("grammar.py: write_total_grammars_file niet gevonden; is Hedy veranderd? Pas pas_hedy_aan.py aan.")
grammar.write_text(s.replace(oud, nieuw), encoding="utf-8")
print("aangepast: grammar.py (schrijven naar grammars-Total mag mislukken)")

for f in map_.rglob("*.py"):
    py_compile.compile(str(f), doraise=True)
print("hedy compileert met", sys.version.split()[0])
