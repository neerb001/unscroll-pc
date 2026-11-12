"""Een tekenschildpad voor de terminal: dezelfde opdrachten als Python's turtle, maar de
tekening verschijnt aan het eind met braille-tekens (elk teken is 2 bij 4 puntjes)."""
import math

BREED, HOOG = 60, 20  # tekens
_lijnen = []  # (x1, y1, x2, y2, kleur)
_KLEUREN = {"black": 37, "zwart": 37, "white": 97, "red": 31, "green": 32, "blue": 34, "yellow": 33,
            "orange": 33, "purple": 35, "pink": 95, "brown": 33, "gray": 90, "grey": 90}


class Turtle:
    def __init__(self, *args, **kwargs):
        self.x = self.y = 0.0
        self.hoek = 0.0
        self.pen = True
        self.kleur = "black"

    def forward(self, afstand):
        nx = self.x + afstand * math.cos(math.radians(self.hoek))
        ny = self.y + afstand * math.sin(math.radians(self.hoek))
        if self.pen:
            _lijnen.append((self.x, self.y, nx, ny, self.kleur))
        self.x, self.y = nx, ny

    fd = forward

    def back(self, afstand):
        self.forward(-afstand)

    backward = bk = back

    def left(self, graden):
        self.hoek += graden

    lt = left

    def right(self, graden):
        self.hoek -= graden

    rt = right

    def setheading(self, graden):
        self.hoek = graden

    def goto(self, x, y=None):
        if y is None:
            x, y = x
        if self.pen:
            _lijnen.append((self.x, self.y, x, y, self.kleur))
        self.x, self.y = x, y

    setpos = setposition = goto

    def penup(self):
        self.pen = False

    pu = up = penup

    def pendown(self):
        self.pen = True

    pd = down = pendown

    def pencolor(self, *kleur):
        if kleur:
            self.kleur = str(kleur[0])

    color = pencolor

    def position(self):
        return (self.x, self.y)

    pos = position

    def heading(self):
        return self.hoek

    def xcor(self):
        return self.x

    def ycor(self):
        return self.y

    def __getattr__(self, naam):  # shape, speed, hideturtle, ... doen hier niets
        return lambda *a, **k: None


def teken():
    if not _lijnen:
        return
    xs = [p for l in _lijnen for p in (l[0], l[2])]
    ys = [p for l in _lijnen for p in (l[1], l[3])]
    minx, maxx, miny, maxy = min(xs), max(xs), min(ys), max(ys)
    bw, bh = BREED * 2 - 1, HOOG * 4 - 1
    schaal = min(bw / max(maxx - minx, 1), bh / max(maxy - miny, 1), 1.0)
    punten = {}
    for x1, y1, x2, y2, kleur in _lijnen:
        stappen = int(max(abs(x2 - x1), abs(y2 - y1)) * schaal) + 1
        for i in range(stappen + 1):
            x = x1 + (x2 - x1) * i / stappen
            y = y1 + (y2 - y1) * i / stappen
            px = int(round((x - minx) * schaal))
            py = int(round((maxy - y) * schaal))
            punten[(px, py)] = kleur
    bits = ((0x01, 0x08), (0x02, 0x10), (0x04, 0x20), (0x40, 0x80))
    hoogte = max(py for _, py in punten) // 4 + 1
    breedte = max(px for px, _ in punten) // 2 + 1
    for rij in range(hoogte):
        regel = ""
        for kol in range(breedte):
            code, kleur = 0, None
            for dy in range(4):
                for dx in range(2):
                    k = punten.get((kol * 2 + dx, rij * 4 + dy))
                    if k is not None:
                        code |= bits[dy][dx]
                        kleur = k
            teken_ = chr(0x2800 + code) if code else " "
            if kleur is not None:
                teken_ = "\x1b[%dm%s\x1b[0m" % (_KLEUREN.get(str(kleur).lower(), 37), teken_)
            regel += teken_
        print(regel.rstrip())
