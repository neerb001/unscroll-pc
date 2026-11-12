"""Maakt hedylang geschikt voor Python 3.11 (Debian 12). Hedy vraagt 3.12, maar gebruikt daarvan
maar twee dingen: een 'type X = ...'-regel en een backslash in een f-string."""
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
for f in map_.rglob("*.py"):
    py_compile.compile(str(f), doraise=True)
print("hedy compileert met", sys.version.split()[0])
