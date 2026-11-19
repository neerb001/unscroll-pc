#!/usr/bin/env python3
"""Maakt de kaarten voor Topografie uit Natural Earth (publiek domein, https://www.naturalearthdata.com).

Gebruik:  python3 -I tools/maak_topo.py NE_MAP app/topo
NE_MAP bevat de geojson-bestanden van https://github.com/nvkelso/natural-earth-vector (map geojson):
ne_50m_admin_0_countries, ne_50m_populated_places_simple, ne_10m_admin_0_countries,
ne_10m_admin_1_states_provinces, ne_10m_rivers_lake_centerlines, ne_10m_lakes,
ne_110m_admin_0_countries, ne_50m_rivers_lake_centerlines.
Schrijft europa.json, nederland.json en wereld.json.

Een kaart heeft een kader, land dat ingekleurd wordt maar niet gevraagd (achtergrond), land
eromheen dat grijs blijft (grijs) en dingen om te leren: vlakken
(landen, provincies, eilanden), punten (steden, zeeën) en lijnen (rivieren). De lijnen worden
vereenvoudigd en afgerond, zodat het bestand klein blijft en een netbook hem snel tekent.
Steden, zeeën en eilanden die niet (goed) in Natural Earth staan, staan hieronder met coördinaten.
"""
import json
import os
import sys


def lees(d, naam):
    with open(os.path.join(d, naam + ".geojson"), encoding="utf-8") as f:
        return json.load(f)["features"]


def ringen_van(geom):
    if not geom:
        return []
    if geom["type"] == "Polygon":
        return [geom["coordinates"][0]]
    if geom["type"] == "MultiPolygon":
        return [p[0] for p in geom["coordinates"]]
    return []


def lijnen_van(geom):
    if not geom:
        return []
    if geom["type"] == "LineString":
        return [geom["coordinates"]]
    if geom["type"] == "MultiLineString":
        return geom["coordinates"]
    return []


def knip(ring, kader, rand):
    """Sutherland-Hodgman: alleen het deel van een ring binnen het (iets ruimere) kader."""
    x0, y0, x1, y1 = kader[0] - rand, kader[1] - rand, kader[2] + rand, kader[3] + rand
    pts = [tuple(p[:2]) for p in ring]
    for binnen, snij in (
        (lambda p: p[0] >= x0, lambda a, b: (x0, a[1] + (b[1] - a[1]) * (x0 - a[0]) / (b[0] - a[0]))),
        (lambda p: p[0] <= x1, lambda a, b: (x1, a[1] + (b[1] - a[1]) * (x1 - a[0]) / (b[0] - a[0]))),
        (lambda p: p[1] >= y0, lambda a, b: (a[0] + (b[0] - a[0]) * (y0 - a[1]) / (b[1] - a[1]), y0)),
        (lambda p: p[1] <= y1, lambda a, b: (a[0] + (b[0] - a[0]) * (y1 - a[1]) / (b[1] - a[1]), y1)),
    ):
        uit = []
        for i, p in enumerate(pts):
            vorig = pts[i - 1]
            if binnen(p):
                if not binnen(vorig):
                    uit.append(snij(vorig, p))
                uit.append(p)
            elif binnen(vorig):
                uit.append(snij(vorig, p))
        pts = uit
        if not pts:
            return []
    return pts


def vereenvoudig(pts, stap, dicht):
    if not pts:
        return None
    uit = [pts[0]]
    for x, y in pts[1:]:
        if abs(x - uit[-1][0]) + abs(y - uit[-1][1]) >= stap:
            uit.append((x, y))
    if dicht and len(uit) < 4:
        return None
    if not dicht and len(uit) < 2:
        uit.append(pts[-1])
    return [[round(x, 3), round(y, 3)] for x, y in uit]


def vlak(ringen, kader, stap, rand=1.0):
    uit = []
    for r in ringen:
        r = vereenvoudig(knip(r, kader, rand), stap, True)
        if r:
            uit.append(r)
    return uit


def lijn(lijnen, kader, stap):
    uit = []
    for l in lijnen:
        if any(kader[0] <= x <= kader[2] and kader[1] <= y <= kader[3] for x, y, *_ in l):
            r = vereenvoudig([tuple(p[:2]) for p in l], stap, False)
            if r:
                uit.append(r)
    return uit


def schrijf(pad, kaart):
    with open(pad, "w", encoding="utf-8") as f:
        json.dump(kaart, f, ensure_ascii=False, separators=(",", ":"))
    n = len(kaart["items"])
    print(f"{pad}: {n} dingen, {os.path.getsize(pad) // 1024} kB")


# ---------------- Europa ----------------

EU_KADER = [-25.0, 34.0, 45.0, 71.5]
EU_EXTRA = {"TUR", "CYP"}  # bij Natural Earth Azië, op school Europa
EU_NIET = {"FRO", "GGY", "JEY", "IMN", "ALD", "AND", "LIE", "MCO", "SMR", "VAT", "MLT"}  # te klein of geen land
EU_STAD_NL = {
    "ALB": "Tirana", "AUT": "Wenen", "BEL": "Brussel", "BGR": "Sofia", "BIH": "Sarajevo", "BLR": "Minsk",
    "CHE": "Bern", "CYP": "Nicosia", "CZE": "Praag", "DEU": "Berlijn", "DNK": "Kopenhagen", "ESP": "Madrid",
    "EST": "Tallinn", "FIN": "Helsinki", "FRA": "Parijs", "GBR": "Londen", "GRC": "Athene", "HRV": "Zagreb",
    "HUN": "Boedapest", "IRL": "Dublin", "ISL": "Reykjavik", "ITA": "Rome", "KOS": "Pristina", "LTU": "Vilnius",
    "LUX": "Luxemburg", "LVA": "Riga", "MDA": "Chisinau", "MKD": "Skopje", "MNE": "Podgorica",
    "NLD": "Amsterdam", "NOR": "Oslo", "POL": "Warschau", "PRT": "Lissabon", "ROU": "Boekarest",
    "RUS": "Moskou", "SRB": "Belgrado", "SVK": "Bratislava", "SVN": "Ljubljana", "SWE": "Stockholm",
    "TUR": "Ankara", "UKR": "Kyiv",
}
EU_STAD_EN = {"AUT": "Vienna", "BEL": "Brussels", "CZE": "Prague", "DEU": "Berlin", "DNK": "Copenhagen",
              "FRA": "Paris", "GBR": "London", "GRC": "Athens", "HUN": "Budapest", "LUX": "Luxembourg",
              "MDA": "Chisinau", "POL": "Warsaw", "PRT": "Lisbon", "ROU": "Bucharest", "RUS": "Moscow",
              "SRB": "Belgrade", "UKR": "Kyiv"}
EU_OOK = {
    "GBR": ["groot-brittannie", "vk", "great britain", "uk", "britain"], "BLR": ["belarus"], "MDA": ["moldova"],
    "MKD": ["macedonie", "macedonia"], "CZE": ["tsjechische republiek", "czech republic"], "BIH": ["bosnie", "bosnia"],
    "NLD": ["holland"],
}
EU_STAD_OOK = {"UKR": ["kiev", "kiew"], "MDA": ["kisjinev"]}


def europa(d):
    steden = {}
    for s in lees(d, "ne_50m_populated_places_simple"):
        p = s["properties"]
        if p.get("adm0cap") == 1 and p["adm0_a3"] not in steden:
            steden[p["adm0_a3"]] = (p["name"], p["longitude"], p["latitude"])
    steden.setdefault("KOS", ("Pristina", 21.17, 42.67))
    achtergrond, grijs, items = [], [], []
    for f in lees(d, "ne_50m_admin_0_countries"):
        p = f["properties"]
        code = p["ADM0_A3"]
        r = vlak(ringen_van(f["geometry"]), EU_KADER, 0.12, 3)
        if not r:
            continue
        europees = p["CONTINENT"] == "Europe" or code in EU_EXTRA
        if not europees:
            grijs.extend(r)
            continue
        if code in EU_NIET:
            achtergrond.extend(r)
            continue
        naam, lon, lat = steden[code]
        items.append({"code": code, "groep": "land", "nl": p["NAME_NL"], "en": p["NAME_EN"], "ook": EU_OOK.get(code, []),
                      "vlak": r, "stad": {"nl": EU_STAD_NL[code], "en": EU_STAD_EN.get(code, naam),
                                          "ook": EU_STAD_OOK.get(code, []), "punt": [round(lon, 3), round(lat, 3)]}})
    items.sort(key=lambda i: i["nl"])
    return {"kader": EU_KADER, "achtergrond": achtergrond, "grijs": grijs, "lijnen": [], "items": items}


# ---------------- Nederland ----------------

NL_KADER = [3.3, 50.72, 7.25, 53.62]
PROVINCIE_STAD = {
    "Groningen": "Groningen", "Friesland": "Leeuwarden", "Drenthe": "Assen", "Overijssel": "Zwolle",
    "Flevoland": "Lelystad", "Gelderland": "Arnhem", "Utrecht": "Utrecht", "Noord-Holland": "Haarlem",
    "Zuid-Holland": "Den Haag", "Zeeland": "Middelburg", "Noord-Brabant": "'s-Hertogenbosch", "Limburg": "Maastricht",
}
PROVINCIE_EN = {"Friesland": "Friesland", "Noord-Holland": "North Holland", "Zuid-Holland": "South Holland",
                "Noord-Brabant": "North Brabant"}
STAD_OOK = {"'s-Hertogenbosch": ["den bosch", "s hertogenbosch", "hertogenbosch"], "Den Haag": ["s-gravenhage", "the hague"]}
# grote steden die op school terugkomen (breedte, lengte)
STEDEN = {
    "Amsterdam": (52.37, 4.90), "Rotterdam": (51.92, 4.48), "Den Haag": (52.08, 4.30), "Utrecht": (52.09, 5.12),
    "Eindhoven": (51.44, 5.48), "Groningen": (53.22, 6.57), "Tilburg": (51.56, 5.09), "Almere": (52.37, 5.21),
    "Breda": (51.59, 4.78), "Nijmegen": (51.84, 5.86), "Apeldoorn": (52.21, 5.97), "Haarlem": (52.38, 4.64),
    "Arnhem": (51.98, 5.91), "Enschede": (52.22, 6.89), "Amersfoort": (52.16, 5.39), "Zaandam": (52.44, 4.83),
    "'s-Hertogenbosch": (51.69, 5.30), "Zwolle": (52.52, 6.09), "Leiden": (52.16, 4.49), "Maastricht": (50.85, 5.69),
    "Dordrecht": (51.81, 4.67), "Leeuwarden": (53.20, 5.80), "Alkmaar": (52.63, 4.75), "Emmen": (52.79, 6.90),
    "Delft": (52.01, 4.36), "Deventer": (52.26, 6.16), "Venlo": (51.37, 6.17), "Hilversum": (52.22, 5.18),
    "Assen": (52.99, 6.56), "Lelystad": (52.52, 5.47), "Middelburg": (51.50, 3.61), "Vlissingen": (51.44, 3.57),
    "Den Helder": (52.96, 4.76), "Gouda": (52.01, 4.71), "Heerlen": (50.89, 5.98), "Roermond": (51.19, 5.99),
    "Hengelo": (52.27, 6.79), "Sneek": (53.03, 5.66), "Harlingen": (53.17, 5.42), "Bergen op Zoom": (51.49, 4.29),
    "Terneuzen": (51.34, 3.83),
}
# zeeën en wateren zonder vlak in Natural Earth: een punt midden op het water
WATER_PUNT = {"Noordzee": ("North Sea", 52.75, 3.9), "Waddenzee": ("Wadden Sea", 53.22, 5.15),
              "Markermeer": ("Markermeer", 52.53, 5.22), "Oosterschelde": ("Eastern Scheldt", 51.58, 3.97),
              "Westerschelde": ("Western Scheldt", 51.40, 3.75)}
RIVIER_NL = {"Rhine": "Rijn", "Ems": "Eems", "Waal": "Waal", "Maas": "Maas", "IJssel": "IJssel", "Lek": "Lek",
             "Nederrijn": "Nederrijn", "Schelde": "Schelde"}
RIVIER_EN = {"Rijn": "Rhine", "Eems": "Ems", "Maas": "Meuse", "Schelde": "Scheldt"}
EILANDEN = {"Texel": (53.07, 4.80), "Vlieland": (53.27, 5.03), "Terschelling": (53.40, 5.33),
            "Ameland": (53.45, 5.77), "Schiermonnikoog": (53.48, 6.20)}


def nederland(d):
    k = NL_KADER
    grijs, items, lijnen = [], [], []
    for f in lees(d, "ne_10m_admin_0_countries"):
        p = f["properties"]
        if p["ADM0_A3"] in ("BEL", "DEU", "FRA", "LUX"):
            grijs.extend(vlak(ringen_van(f["geometry"]), k, 0.02, 2.5))  # ruim: ook ingezoomd breed genoeg
        if p["ADM0_A3"] == "NLD":
            # de Waddeneilanden: de losse stukken land boven de kust
            for ring in ringen_van(f["geometry"]):
                xs, ys = [q[0] for q in ring], [q[1] for q in ring]
                mx, my = sum(xs) / len(xs), sum(ys) / len(ys)
                for naam, (lat, lon) in EILANDEN.items():
                    if abs(my - lat) < 0.12 and abs(mx - lon) < 0.25 and len(ring) > 5:
                        items.append({"code": "eiland:" + naam, "groep": "water", "soort": "eiland", "nl": naam, "en": naam,
                                      "vlak": vlak([ring], k, 0.008)})
    for f in lees(d, "ne_10m_admin_1_states_provinces"):
        p = f["properties"]
        if p["adm0_a3"] != "NLD" or p["name_nl"] not in PROVINCIE_STAD:
            continue
        naam = p["name_nl"]
        stad = PROVINCIE_STAD[naam]
        lat, lon = STEDEN[stad]
        items.append({"code": "prov:" + naam, "groep": "provincie", "nl": naam, "en": PROVINCIE_EN.get(naam, naam),
                      "vlak": vlak(ringen_van(f["geometry"]), k, 0.015),
                      "stad": {"nl": stad, "en": "The Hague" if stad == "Den Haag" else stad, "ook": STAD_OOK.get(stad, []),
                               "punt": [lon, lat]}})
    for naam, (lat, lon) in STEDEN.items():
        items.append({"code": "stad:" + naam, "groep": "stad", "nl": naam, "en": "The Hague" if naam == "Den Haag" else naam,
                      "ook": STAD_OOK.get(naam, []), "punt": [lon, lat]})
    rivieren = {}
    for f in lees(d, "ne_10m_rivers_lake_centerlines"):
        naam = RIVIER_NL.get(f["properties"].get("name") or "")
        if naam:
            l = lijn(lijnen_van(f["geometry"]), k, 0.01)
            if l:
                rivieren.setdefault(naam, []).extend(l)
                lijnen.extend(l)
    for naam, l in sorted(rivieren.items()):
        items.append({"code": "rivier:" + naam, "groep": "water", "soort": "rivier", "nl": naam, "en": RIVIER_EN.get(naam, naam), "lijn": l})
    for f in lees(d, "ne_10m_lakes"):
        if f["properties"].get("name") == "IJsselmeer":
            r = vlak(ringen_van(f["geometry"]), k, 0.01)
            items.append({"code": "water:IJsselmeer", "groep": "water", "nl": "IJsselmeer", "en": "IJsselmeer", "vlak": r})
    for naam, (en, lat, lon) in WATER_PUNT.items():
        items.append({"code": "water:" + naam, "groep": "water", "nl": naam, "en": en, "punt": [lon, lat]})
    return {"kader": k, "achtergrond": [], "grijs": grijs, "lijnen": lijnen, "items": items}


# ---------------- Wereld (groep 8) ----------------

W_KADER = [-180.0, -57.0, 180.0, 83.0]
WERELDDEEL = {"Africa": ("Afrika", "Africa"), "Asia": ("Azië", "Asia"), "Europe": ("Europa", "Europe"),
              "North America": ("Noord-Amerika", "North America"), "South America": ("Zuid-Amerika", "South America"),
              "Oceania": ("Oceanië", "Oceania")}
WERELDDEEL_OOK = {"Oceania": ["australie", "australia"], "North America": ["amerika"], "South America": ["latijns-amerika"]}
# landen die in groep 8 terugkomen: code, hoofdstad (nl, en), lengte, breedte
W_LANDEN = {
    "USA": ("Washington", "Washington", -77.04, 38.90), "CAN": ("Ottawa", "Ottawa", -75.70, 45.42),
    "MEX": ("Mexico-Stad", "Mexico City", -99.13, 19.43), "BRA": ("Brasilia", "Brasília", -47.88, -15.79),
    "ARG": ("Buenos Aires", "Buenos Aires", -58.38, -34.60), "CHL": ("Santiago", "Santiago", -70.67, -33.45),
    "PER": ("Lima", "Lima", -77.04, -12.05), "COL": ("Bogota", "Bogotá", -74.07, 4.71),
    "VEN": ("Caracas", "Caracas", -66.90, 10.49), "SUR": ("Paramaribo", "Paramaribo", -55.20, 5.85),
    "CUB": ("Havana", "Havana", -82.37, 23.11), "EGY": ("Caïro", "Cairo", 31.24, 30.04),
    "MAR": ("Rabat", "Rabat", -6.84, 34.02), "DZA": ("Algiers", "Algiers", 3.06, 36.75),
    "NGA": ("Abuja", "Abuja", 7.49, 9.08), "ZAF": ("Pretoria", "Pretoria", 28.19, -25.75),
    "KEN": ("Nairobi", "Nairobi", 36.82, -1.29), "ETH": ("Addis Abeba", "Addis Ababa", 38.75, 9.03),
    "COD": ("Kinshasa", "Kinshasa", 15.31, -4.32), "MDG": ("Antananarivo", "Antananarivo", 47.52, -18.88),
    "RUS": ("Moskou", "Moscow", 37.62, 55.76), "CHN": ("Peking", "Beijing", 116.40, 39.90),
    "IND": ("New Delhi", "New Delhi", 77.21, 28.61), "JPN": ("Tokio", "Tokyo", 139.69, 35.69),
    "IDN": ("Jakarta", "Jakarta", 106.85, -6.21), "AUS": ("Canberra", "Canberra", 149.13, -35.28),
    "NZL": ("Wellington", "Wellington", 174.78, -41.29), "SAU": ("Riyad", "Riyadh", 46.68, 24.71),
    "IRN": ("Teheran", "Tehran", 51.39, 35.69), "IRQ": ("Bagdad", "Baghdad", 44.37, 33.31),
    "TUR": ("Ankara", "Ankara", 32.85, 39.93), "PAK": ("Islamabad", "Islamabad", 73.05, 33.68),
    "AFG": ("Kabul", "Kabul", 69.21, 34.56), "THA": ("Bangkok", "Bangkok", 100.50, 13.76),
    "VNM": ("Hanoi", "Hanoi", 105.85, 21.03), "PHL": ("Manilla", "Manila", 120.98, 14.60),
    "KOR": ("Seoul", "Seoul", 126.98, 37.57), "MNG": ("Ulaanbaatar", "Ulaanbaatar", 106.91, 47.89),
    "KAZ": ("Astana", "Astana", 71.45, 51.17), "GRL": ("Nuuk", "Nuuk", -51.72, 64.18),
    "GBR": ("Londen", "London", -0.13, 51.51), "FRA": ("Parijs", "Paris", 2.35, 48.86),
    "DEU": ("Berlijn", "Berlin", 13.40, 52.52), "ESP": ("Madrid", "Madrid", -3.70, 40.42),
    "ITA": ("Rome", "Rome", 12.50, 41.90), "NOR": ("Oslo", "Oslo", 10.75, 59.91),
}
W_LAND_OOK = {"USA": ["amerika", "vs", "verenigde staten van amerika", "usa"], "COD": ["congo", "dr congo"],
              "GBR": ["groot-brittannie", "engeland", "vk", "uk"], "KOR": ["korea"], "SAU": ["saudi-arabie"]}
W_NAAM = {"CHN": "China", "USA": "Verenigde Staten", "COD": "Congo"}  # zoals op school, niet de officiële naam
W_STAD_OOK = {"CHN": ["beijing"], "JPN": ["tokyo"], "COL": ["bogotá"], "BRA": ["brasília"], "PHL": ["manila"],
              "EGY": ["cairo", "kairo"], "SAU": ["riyadh"], "MEX": ["mexico city", "mexico"], "ETH": ["addis ababa"],
              "USA": ["washington dc", "washington d.c."], "IRN": ["tehran"], "IRQ": ["baghdad"], "CUB": ["havana", "havanna"]}
W_NATUUR = {  # punten midden in de zee, op het gebergte of in de woestijn
    "Grote Oceaan": ("Pacific Ocean", -140, 5, ["stille oceaan", "pacific"]),
    "Atlantische Oceaan": ("Atlantic Ocean", -35, 20, ["atlantic"]),
    "Indische Oceaan": ("Indian Ocean", 78, -22, ["indian ocean"]),
    "Noordelijke IJszee": ("Arctic Ocean", 0, 80, ["arctic ocean"]),
    "Zuidelijke Oceaan": ("Southern Ocean", 20, -55, ["southern ocean"]),
    "Middellandse Zee": ("Mediterranean Sea", 18, 34.5, ["mediterranean"]),
    "Caribische Zee": ("Caribbean Sea", -75, 15, ["caribbean"]),
    "Rode Zee": ("Red Sea", 38.5, 20, ["red sea"]),
    "Himalaya": ("Himalayas", 84, 28.5, ["himalayas"]),
    "Andes": ("Andes", -69, -20, []),
    "Rocky Mountains": ("Rocky Mountains", -110, 44, ["rockies"]),
    "Sahara": ("Sahara", 10, 23, []),
}
W_RIVIER = {"Nijl": ("Nile", {"Nile", "El Bahr el Abyad", "Victoria Nile", "Albert Nile", "Bahr el Jebel", "Rosetta Branch"}),
            "Amazone": ("Amazon", {"Amazonas", "Ucayali"}),
            "Mississippi": ("Mississippi", {"Mississippi"}),
            "Jangtsekiang": ("Yangtze", {"Yangtze", "Chang Jiang", "Jinsha", "Tongtian"}),
            "Congo": ("Congo", {"Congo", "Lualaba"}), "Mekong": ("Mekong", {"Mekong", "Lancang"}),
            "Donau": ("Danube", {"Danube", "Donau"})}
W_RIVIER_OOK = {"Jangtsekiang": ["jangtse", "yangtze"], "Amazone": ["amazon"], "Nijl": ["nile"]}


def wereld(d):
    k = W_KADER
    achtergrond, items = [], []
    delen = {}
    for f in lees(d, "ne_110m_admin_0_countries"):
        p = f["properties"]
        code = p["ADM0_A3"]
        r = vlak(ringen_van(f["geometry"]), k, 0.4, 0)
        if not r:
            continue
        deel = p["CONTINENT"]
        if code == "RUS":
            deel = "Asia"  # het grootste stuk; Europa staat op de eigen kaart
        if deel in WERELDDEEL:
            delen.setdefault(deel, []).extend(r)
        if code in W_LANDEN:
            snl, sen, lon, lat = W_LANDEN[code]
            items.append({"code": "land:" + code, "groep": "land", "nl": W_NAAM.get(code, p["NAME_NL"]), "en": p["NAME_EN"],
                          "ook": W_LAND_OOK.get(code, []) + [p["NAME_NL"]], "vlak": r,
                          "stad": {"nl": snl, "en": sen, "ook": W_STAD_OOK.get(code, []), "punt": [lon, lat]}})
        else:
            achtergrond.extend(r)
    for deel, r in delen.items():
        nl, en = WERELDDEEL[deel]
        items.append({"code": "deel:" + deel, "groep": "werelddeel", "nl": nl, "en": en,
                      "ook": WERELDDEEL_OOK.get(deel, []), "vlak": r})
    rivieren = {}
    for f in lees(d, "ne_50m_rivers_lake_centerlines"):
        naam = f["properties"].get("name")
        for nl, (en, namen) in W_RIVIER.items():
            if naam in namen:
                rivieren.setdefault(nl, []).extend(lijn(lijnen_van(f["geometry"]), k, 0.3))
    for nl, l in rivieren.items():
        items.append({"code": "rivier:" + nl, "groep": "natuur", "nl": nl, "en": W_RIVIER[nl][0],
                      "ook": W_RIVIER_OOK.get(nl, []), "lijn": l})
    for nl, (en, lon, lat, ook) in W_NATUUR.items():
        items.append({"code": "natuur:" + nl, "groep": "natuur", "nl": nl, "en": en, "ook": ook, "punt": [lon, lat]})
    return {"kader": k, "achtergrond": achtergrond, "grijs": [], "lijnen": [r for i in items if "lijn" in i for r in i["lijn"]], "items": items}


d, uit = sys.argv[1], sys.argv[2]
os.makedirs(uit, exist_ok=True)
schrijf(os.path.join(uit, "europa.json"), europa(d))
schrijf(os.path.join(uit, "nederland.json"), nederland(d))
schrijf(os.path.join(uit, "wereld.json"), wereld(d))
