package main

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Computerlab: Linux leren in een nagebootste terminal. Er draait niets echt; het kind kan
// niets kapotmaken en komt nooit bij het echte systeem. Missies zonder punten of tijd.

type labWereld struct {
	Mappen    map[string]bool   `json:"mappen"`
	Bestanden map[string]string `json:"bestanden"`
	Cwd       string            `json:"cwd"`
	Missie    int               `json:"missie"`
}

const labThuis = "/thuis"

func nieuweLabWereld() *labWereld {
	w := &labWereld{Cwd: labThuis, Mappen: map[string]bool{}, Bestanden: map[string]string{}}
	for _, m := range []string{"/", labThuis, labThuis + "/tuin", labThuis + "/tuin/schuur", labThuis + "/zolder", labThuis + "/zolder/doos"} {
		w.Mappen[m] = true
	}
	w.Bestanden[labThuis+"/tuin/briefje.txt"] = "briefje"
	w.Bestanden[labThuis+"/tuin/schuur/sleutel.txt"] = "sleutel"
	w.Bestanden[labThuis+"/tuin/schuur/hark.txt"] = "hark"
	w.Bestanden[labThuis+"/zolder/doos/kaart.txt"] = "kaart"
	return w
}

func labPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "lab.json") }

func laadLab() *labWereld {
	w := nieuweLabWereld()
	if data, err := os.ReadFile(labPad()); err == nil {
		var g labWereld
		if json.Unmarshal(data, &g) == nil && g.Mappen != nil && g.Bestanden != nil && g.Cwd != "" {
			return &g
		}
	}
	return w
}

func (w *labWereld) bewaar() {
	_ = os.MkdirAll(filepath.Dir(labPad()), 0o755)
	data, _ := json.Marshal(w)
	_ = os.WriteFile(labPad(), data, 0o644)
}

func (w *labWereld) volledig(p string) string {
	if p == "" || p == "~" {
		return labThuis
	}
	if strings.HasPrefix(p, "~/") {
		p = labThuis + p[1:]
	}
	if !strings.HasPrefix(p, "/") {
		p = w.Cwd + "/" + p
	}
	return path.Clean(p)
}

func (w *labWereld) kort(p string) string {
	if p == labThuis {
		return "~"
	}
	if strings.HasPrefix(p, labThuis+"/") {
		return "~" + p[len(labThuis):]
	}
	return p
}

// Speciale inhoud: verwijzingen naar teksten in de taal van het kind.
var labTeksten = map[string][2]string{
	"briefje": {"Psst! In de schuur ligt een sleutel. Ga ernaartoe met cd schuur en kijk rond met ls.",
		"Psst! There is a key in the shed. Go there with cd schuur and look around with ls."},
	"sleutel": {"Een oude koperen sleutel. Er hangt een label aan: 'zolder'.",
		"An old brass key. A label says: 'zolder' (attic)."},
	"hark": {"Gewoon een hark. Niks bijzonders.", "Just a rake. Nothing special."},
	"kaart": {"Een schatkaart! Je hebt alle missies in de tuin en op zolder gevonden. Knap gedaan.",
		"A treasure map! You found everything in the garden and the attic. Well done."},
}

func (w *labWereld) inhoud(p, taal string) string {
	t := w.Bestanden[p]
	if x, ok := labTeksten[t]; ok {
		if taal == "nl" {
			return x[0]
		}
		return x[1]
	}
	return t
}

func (w *labWereld) kinderen(map_ string) []string {
	var uit []string
	for m := range w.Mappen {
		if m != map_ && path.Dir(m) == map_ {
			uit = append(uit, path.Base(m)+"/")
		}
	}
	for b := range w.Bestanden {
		if path.Dir(b) == map_ {
			uit = append(uit, path.Base(b))
		}
	}
	sort.Strings(uit)
	return uit
}

// Missie: wat het kind moet doen en hoe we zien dat het gelukt is.
type labMissie struct {
	nl, en string
	klaar  func(w *labWereld, cmd string, args []string) bool
}

var labMissies = []labMissie{
	{"Waar ben je? Typ pwd (print working directory) om te zien in welke map je bent.",
		"Where are you? Type pwd (print working directory) to see which folder you are in.",
		func(w *labWereld, c string, a []string) bool { return c == "pwd" }},
	{"Kijk wat er in je map staat met ls (list). Mappen eindigen op een /.",
		"See what is in your folder with ls (list). Folders end with a /.",
		func(w *labWereld, c string, a []string) bool { return c == "ls" }},
	{"Ga naar de map tuin met cd tuin (change directory).",
		"Go into the folder tuin (garden) with cd tuin (change directory).",
		func(w *labWereld, c string, a []string) bool { return w.Cwd == labThuis+"/tuin" }},
	{"Er ligt een briefje. Lees het met cat briefje.txt.",
		"There is a note. Read it with cat briefje.txt.",
		func(w *labWereld, c string, a []string) bool {
			return c == "cat" && len(a) > 0 && w.volledig(a[0]) == labThuis+"/tuin/briefje.txt"
		}},
	{"Volg het briefje: vind de sleutel en lees wat erop staat.",
		"Follow the note: find the key and read what it says.",
		func(w *labWereld, c string, a []string) bool {
			return c == "cat" && len(a) > 0 && strings.HasSuffix(w.volledig(a[0]), "/sleutel.txt")
		}},
	{"Neem de sleutel mee naar huis: mv sleutel.txt ~ (de ~ is je thuismap).",
		"Take the key home: mv sleutel.txt ~ (the ~ is your home folder).",
		func(w *labWereld, c string, a []string) bool {
			_, ok := w.Bestanden[labThuis+"/sleutel.txt"]
			return ok
		}},
	{"Ga terug naar huis met cd (zonder iets erachter) of cd ~. Met cd .. ga je één map omhoog.",
		"Go back home with cd (nothing after it) or cd ~. With cd .. you go up one folder.",
		func(w *labWereld, c string, a []string) bool { return w.Cwd == labThuis }},
	{"Maak een eigen map: mkdir uitvindingen (make directory).",
		"Make your own folder: mkdir uitvindingen (make directory).",
		func(w *labWereld, c string, a []string) bool { return w.Mappen[labThuis+"/uitvindingen"] }},
	{"Schrijf een idee op: echo een robot die de hond uitlaat > uitvindingen/robot.txt",
		"Write down an idea: echo a robot that walks the dog > uitvindingen/robot.txt",
		func(w *labWereld, c string, a []string) bool {
			return strings.TrimSpace(w.Bestanden[labThuis+"/uitvindingen/robot.txt"]) != ""
		}},
	{"Op zolder staat een doos. Ga erheen en vind wat erin zit. (Tip: cd zolder/doos kan in één keer.)",
		"There is a box in the attic (zolder). Go there and find what is inside. (Tip: cd zolder/doos works in one go.)",
		func(w *labWereld, c string, a []string) bool {
			return c == "cat" && len(a) > 0 && strings.HasSuffix(w.volledig(a[0]), "/kaart.txt")
		}},
	{"De hark heb je niet nodig. Verwijder hem met rm tuin/schuur/hark.txt (rm = remove, weg is weg!).",
		"You don't need the rake. Remove it with rm tuin/schuur/hark.txt (rm = remove, gone is gone!).",
		func(w *labWereld, c string, a []string) bool {
			_, ok := w.Bestanden[labThuis+"/tuin/schuur/hark.txt"]
			return !ok
		}},
}

// voerUit voert één opdracht uit en geeft de uitvoer.
func (w *labWereld) voerUit(regel, taal string) (string, bool) {
	nl := taal == "nl"
	t := func(a, b string) string {
		if nl {
			return a
		}
		return b
	}
	regel = strings.TrimSpace(regel)
	if regel == "" {
		return "", false
	}
	// echo tekst > bestand
	if strings.HasPrefix(regel, "echo ") || regel == "echo" {
		tekst, doel, omleiden := strings.Cut(strings.TrimPrefix(regel, "echo"), ">")
		tekst = strings.Trim(strings.TrimSpace(tekst), `"'`)
		if !omleiden {
			return tekst, w.check("echo", nil)
		}
		p := w.volledig(strings.TrimSpace(doel))
		if !w.Mappen[path.Dir(p)] || w.Mappen[p] {
			return t("Die map bestaat niet.", "That folder does not exist."), false
		}
		w.Bestanden[p] = tekst
		return "", w.check("echo", []string{doel})
	}
	delen := strings.Fields(regel)
	cmd, args := delen[0], delen[1:]
	uit := ""
	switch cmd {
	case "help":
		uit = t("Opdrachten: pwd, ls, cd, cat, mkdir, touch, echo, mv, rm, clear, missie, opnieuw",
			"Commands: pwd, ls, cd, cat, mkdir, touch, echo, mv, rm, clear, missie (mission), opnieuw (start over)")
	case "pwd":
		uit = w.Cwd
	case "ls":
		doel := w.Cwd
		if len(args) > 0 {
			doel = w.volledig(args[0])
		}
		if !w.Mappen[doel] {
			return t("ls: die map bestaat niet", "ls: no such folder"), false
		}
		uit = strings.Join(w.kinderen(doel), "  ")
	case "cd":
		doel := labThuis
		if len(args) > 0 {
			doel = w.volledig(args[0])
		}
		if !w.Mappen[doel] {
			if _, isBestand := w.Bestanden[doel]; isBestand {
				return t("cd: dat is een bestand, geen map", "cd: that is a file, not a folder"), false
			}
			return t("cd: die map bestaat niet", "cd: no such folder"), false
		}
		w.Cwd = doel
	case "cat":
		if len(args) == 0 {
			return t("cat: welk bestand?", "cat: which file?"), false
		}
		p := w.volledig(args[0])
		if _, ok := w.Bestanden[p]; !ok {
			if w.Mappen[p] {
				return t("cat: dat is een map. Kijk erin met ls.", "cat: that is a folder. Look inside with ls."), false
			}
			return t("cat: dat bestand bestaat niet", "cat: no such file"), false
		}
		uit = w.inhoud(p, taal)
	case "mkdir", "touch":
		if len(args) == 0 {
			return cmd + t(": welke naam?", ": which name?"), false
		}
		p := w.volledig(args[0])
		if !w.Mappen[path.Dir(p)] {
			return cmd + t(": die map bestaat niet", ": no such folder"), false
		}
		if cmd == "mkdir" {
			w.Mappen[p] = true
		} else if _, ok := w.Bestanden[p]; !ok {
			w.Bestanden[p] = ""
		}
	case "mv":
		if len(args) < 2 {
			return t("mv: gebruik mv wat waarheen", "mv: use mv what where"), false
		}
		van, naar := w.volledig(args[0]), w.volledig(args[1])
		inhoud, ok := w.Bestanden[van]
		if !ok {
			return t("mv: dat bestand bestaat niet (mappen verplaatsen kan hier niet)", "mv: no such file (moving folders is not possible here)"), false
		}
		if w.Mappen[naar] {
			naar = naar + "/" + path.Base(van)
		}
		if !w.Mappen[path.Dir(naar)] {
			return t("mv: die map bestaat niet", "mv: no such folder"), false
		}
		delete(w.Bestanden, van)
		w.Bestanden[naar] = inhoud
	case "rm":
		if len(args) == 0 {
			return t("rm: welk bestand?", "rm: which file?"), false
		}
		p := w.volledig(args[0])
		if _, ok := w.Bestanden[p]; !ok {
			return t("rm: dat bestand bestaat niet (mappen weghalen kan hier niet)", "rm: no such file (removing folders is not possible here)"), false
		}
		delete(w.Bestanden, p)
	case "missie", "mission":
		return w.missieTekst(taal), false
	default:
		return cmd + t(": die opdracht ken ik niet. Typ help.", ": unknown command. Type help."), false
	}
	return uit, w.check(cmd, args)
}

func (w *labWereld) check(cmd string, args []string) bool {
	if w.Missie < len(labMissies) && labMissies[w.Missie].klaar(w, cmd, args) {
		w.Missie++
		return true
	}
	return false
}

func (w *labWereld) missieTekst(taal string) string {
	if w.Missie >= len(labMissies) {
		if taal == "nl" {
			return "Alle missies gedaan! Je kunt vrij rondkijken, of opnieuw beginnen met opnieuw."
		}
		return "All missions done! Look around freely, or start over with opnieuw."
	}
	m := labMissies[w.Missie]
	if taal == "nl" {
		return m.nl
	}
	return m.en
}
