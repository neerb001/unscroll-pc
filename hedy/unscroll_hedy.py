#!/usr/bin/env python3
"""Unscroll: een Hedy-programma omzetten naar Python en uitvoeren in de terminal.

Gebruik:  python3 -I unscroll_hedy.py NIVEAU TAAL BESTAND

Hedy (https://hedy.org, EUPL-1.2) zet de code om; daarna draait het programma met een beperkte
set ingebouwde functies, een tijdslimiet en een tekenschildpad die met tekens tekent.
Een fout komt als één regel met FOUT ervoor en JSON erachter, zodat de app hem netjes kan tonen.
"""
import builtins
import json
import os
import resource
import sys

HIER = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HIER, "lib"))
sys.path.insert(0, os.path.join(HIER, "schildpad"))  # onze turtle.py gaat voor
FOUT = "\x1eHEDYFOUT "
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
    if res.has_pressed or res.has_music:
        meld_fout({"nl": "Toetsen indrukken en muziek werken nog niet op deze computer."}.get(
            taal, "Key presses and music do not work on this computer yet."))

    prefix = ""
    for naam in ["normal"] + (["turtle"] if res.has_turtle else []):
        with open(os.path.join(HIER, "lib", "hedy", "prefixes", naam + ".py"), encoding="utf-8") as f:
            prefix += f.read() + "\n"

    # grenzen: rekentijd en geheugen (wachten op een antwoord telt niet als rekentijd)
    resource.setrlimit(resource.RLIMIT_CPU, (20, 25))
    resource.setrlimit(resource.RLIMIT_AS, (300 << 20, 300 << 20))

    veilig = {k: v for k, v in vars(builtins).items()
              if k not in ("open", "exec", "eval", "compile", "breakpoint", "exit", "quit", "help")}
    veilig["__import__"] = veilig_importeren
    omgeving = {"__name__": "__main__", "__builtins__": veilig}
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
