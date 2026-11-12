#!/usr/bin/env python3
"""Unscroll: een Hedy-programma omzetten naar Python en uitvoeren in de terminal.

Gebruik:  python3 -I unscroll_hedy.py NIVEAU TAAL BESTAND

Hedy (https://hedy.org, EUPL-1.2) zet de code om; daarna draait het programma met een beperkte
set ingebouwde functies, een tijdslimiet en een tekenschildpad die met tekens tekent.
Een fout komt als één regel met FOUT ervoor en JSON erachter, zodat de app hem netjes kan tonen.
"""
import builtins
import json
import math
import os
import re
import resource
import struct
import subprocess
import sys
import types

HIER = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HIER, "lib"))
sys.path.insert(0, os.path.join(HIER, "schildpad"))  # onze turtle.py gaat voor
FOUT = "\x1eHEDYFOUT "
TOETS = "\x1eHEDYTOETS"  # de app stuurt dan meteen de volgende toets, zonder Enter
WIS = "\x1eHEDYWIS"      # de app maakt het uitvoerscherm leeg
TOEGESTAAN = {"random", "time", "math", "turtle", "extensions"}


class _Leeg(dict):
    def __missing__(self, sleutel):
        return ""


def meld_fout(tekst, regel=None, code=2):
    print(FOUT + json.dumps({"tekst": tekst, "regel": regel}), flush=True)
    sys.exit(code)


def fouttekst(ex, taal):
    try:
        with open(os.path.join(HIER, "fouten.json"), encoding="utf-8") as f:
            fouten = json.load(f)
    except OSError:
        fouten = {}
    code = getattr(ex, "error_code", "") or type(ex).__name__
    sjabloon = fouten.get(taal, {}).get(code) or fouten.get("en", {}).get(code) or str(ex) or code
    args = {k: str(v) for k, v in (getattr(ex, "arguments", None) or {}).items()}
    try:
        return sjabloon.format_map(_Leeg(args)), args.get("line_number")
    except (ValueError, IndexError):
        return sjabloon, args.get("line_number")


def maak_extensions(omgeving):
    """Wat op hedy.org de browser doet: toetsen (if ... is pressed) en het scherm wissen (clear)."""
    ext = types.ModuleType("extensions")

    def if_pressed(mapping):
        print(TOETS, flush=True)
        toets = sys.stdin.readline().rstrip("\n")
        naam = mapping.get(toets) or mapping.get(toets.lower()) or mapping.get(toets.upper()) or mapping.get("else")
        functie = omgeving.get(naam)
        if callable(functie):
            functie()

    def clear():
        print(WIS, flush=True)

    ext.if_pressed = if_pressed
    ext.clear = clear
    return ext


HALVE_TONEN = {"C": -9, "D": -7, "E": -5, "F": -4, "G": -2, "A": 0, "B": 2}


def toonhoogte(noot):
    m = re.fullmatch(r"([A-G])(#|B)?(\d)", str(noot).upper())
    if not m:
        return None
    halve = HALVE_TONEN[m.group(1)] + {"#": 1, "B": -1}.get(m.group(2) or "", 0) + (int(m.group(3)) - 4) * 12
    return 440.0 * 2 ** (halve / 12)


def speel(noot):
    """Speelt een noot van 0,4 seconde via aplay (ALSA). Zonder geluid gebeurt er gewoon niets."""
    f = toonhoogte(noot)
    if not f:
        return
    rate, duur = 22050, 0.4
    n = int(rate * duur)
    golf = b"".join(struct.pack("<h", int(12000 * math.sin(2 * math.pi * f * i / rate) * min(1, (n - i) / 800)))
                    for i in range(n))
    kop = b"RIFF" + struct.pack("<I", 36 + len(golf)) + b"WAVEfmt " + struct.pack("<IHHIIHH", 16, 1, 1, rate, rate * 2, 2, 16) \
        + b"data" + struct.pack("<I", len(golf))
    try:
        p = subprocess.Popen(["aplay", "-q", "-"], stdin=subprocess.PIPE, stderr=subprocess.DEVNULL)
        p.stdin.write(kop + golf)
        p.stdin.close()
    except OSError:
        pass


def veilig_importeren(naam, *args, **kwargs):
    if naam.split(".")[0] not in TOEGESTAAN:
        raise ImportError(naam)
    return _echt_importeren(naam, *args, **kwargs)


_echt_importeren = builtins.__import__


def main():
    if len(sys.argv) != 4:
        print(__doc__)
        sys.exit(1)
    niveau, taal, pad = int(sys.argv[1]), sys.argv[2], sys.argv[3]
    with open(pad, encoding="utf-8") as f:
        code = f.read()
    import hedy

    try:
        res = hedy.transpile(code, niveau, taal)
    except Exception as ex:  # noqa: BLE001 - elke fout van Hedy naar het kind
        tekst, regel = fouttekst(ex, taal)
        meld_fout(tekst, regel)
    prefix = ""
    delen = ["normal"] + (["turtle"] if res.has_turtle else []) + (["pressed"] if res.has_pressed else []) \
        + (["music"] if res.has_music else [])
    for naam in delen:
        with open(os.path.join(HIER, "lib", "hedy", "prefixes", naam + ".py"), encoding="utf-8") as f:
            prefix += f.read() + "\n"

    # grenzen: rekentijd en geheugen (wachten op een antwoord telt niet als rekentijd)
    resource.setrlimit(resource.RLIMIT_CPU, (20, 25))
    resource.setrlimit(resource.RLIMIT_AS, (300 << 20, 300 << 20))

    veilig = {k: v for k, v in vars(builtins).items()
              if k not in ("open", "exec", "eval", "compile", "breakpoint", "exit", "quit", "help")}
    veilig["__import__"] = veilig_importeren
    omgeving = {"__name__": "__main__", "__builtins__": veilig, "play": speel}
    sys.modules["extensions"] = maak_extensions(omgeving)
    try:
        exec(compile(prefix + res.code, "<hedy>", "exec"), omgeving)
    except KeyboardInterrupt:
        pass
    except Exception as ex:  # noqa: BLE001 - fouten tijdens het draaien
        tekst, regel = fouttekst(ex, taal)
        meld_fout(tekst, regel, 3)
    finally:
        if res.has_turtle:
            import turtle
            turtle.teken()


if __name__ == "__main__":
    main()
