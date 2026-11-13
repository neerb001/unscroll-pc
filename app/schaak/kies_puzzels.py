import csv, gzip, json, random, sys
random.seed(7)
banden = [(400, 900), (900, 1200), (1200, 1500), (1500, 1900)]
per = 500
gekozen = {i: [] for i in range(len(banden))}
with open(sys.argv[1], newline="") as f:
    for r in csv.DictReader(f):
        try:
            rating, dev, pop, plays = int(r["Rating"]), int(r["RatingDeviation"]), int(r["Popularity"]), int(r["NbPlays"])
        except ValueError:
            continue
        zetten = r["Moves"].split()
        if pop < 85 or plays < 800 or dev > 90 or len(zetten) > 8:
            continue
        for i, (lo, hi) in enumerate(banden):
            if lo <= rating < hi:
                gekozen[i].append({"f": r["FEN"], "m": r["Moves"], "r": rating, "t": r["Themes"].split()})
uit = []
for i in range(len(banden)):
    lijst = gekozen[i]
    random.shuffle(lijst)
    print("niveau", i, "beschikbaar", len(lijst), file=sys.stderr)
    for p in lijst[:per]:
        p["n"] = i
        uit.append(p)
data = json.dumps(uit, separators=(",", ":")).encode()
with gzip.open(sys.argv[2], "wb", 9) as g:
    g.write(data)
print("puzzels", len(uit), "bytes", len(data), file=sys.stderr)
