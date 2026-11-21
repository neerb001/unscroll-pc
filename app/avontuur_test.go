package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"math/rand/v2"
	"path"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Elk avontuur klopt: elke keuze leidt ergens heen, alles is bereikbaar, er zijn eindes, en
// elke puzzel is op te lossen. Zo kan een tikfout in een verhaal nooit een kind laten vastlopen.
func TestAlleAvonturenKloppen(t *testing.T) {
	namen, _ := avonturenBestanden.ReadDir("avonturen")
	if len(namen) == 0 {
		t.Fatal("geen avonturen")
	}
	oplossingRe := regexp.MustCompile(`(?m)^//\s*oplossing\s+(\S+):\s*(.+)$`)
	for _, n := range namen {
		data, _ := avonturenBestanden.ReadFile(path.Join("avonturen", n.Name()))
		a, err := leesAvontuur(strings.TrimSuffix(n.Name(), ".txt"), string(data))
		if err != nil {
			t.Errorf("%s: %v", n.Name(), err)
			continue
		}
		bestaat := func(waar, id string) {
			if a.scenes[id] == nil {
				t.Errorf("%s: %s verwijst naar scène %q, die bestaat niet", n.Name(), waar, id)
			}
		}
		// alles bereikbaar vanaf het begin
		bereikt := map[string]bool{}
		werk := []string{a.begin}
		for len(werk) > 0 {
			id := werk[0]
			werk = werk[1:]
			if bereikt[id] || a.scenes[id] == nil {
				continue
			}
			bereikt[id] = true
			sc := a.scenes[id]
			for _, k := range sc.keuzes {
				werk = append(werk, k.doel)
			}
			if sc.naarGoed != "" {
				werk = append(werk, sc.naarGoed)
			}
			if sc.anders != nil {
				werk = append(werk, sc.anders.doel)
			}
		}
		for id, sc := range a.scenes {
			if !bereikt[id] {
				t.Errorf("%s: scène %q is niet te bereiken", n.Name(), id)
			}
			for _, k := range sc.keuzes {
				bestaat(id, k.doel)
			}
			invoer := sc.vraag != "" || sc.robot != ""
			if invoer {
				bestaat(id, sc.naarGoed)
				if sc.naarFout != "" {
					t.Errorf("%s: scène %q: geen fout-scène meer, zet de hulp in tip:-regels", n.Name(), id)
				}
				if len(sc.tips) < 2 {
					t.Errorf("%s: scène %q: minstens twee tips nodig (van zacht naar sterk)", n.Name(), id)
				}
				for _, tp := range sc.tips {
					if strings.Contains(tp, "—") {
						t.Errorf("%s: scène %q: gedachtestreepje in een tip", n.Name(), id)
					}
				}
				if sc.anders != nil {
					bestaat(id, sc.anders.doel)
				}
				if len(sc.keuzes) > 0 {
					t.Errorf("%s: scène %q heeft een vraag én keuzes (cijfers zijn dan antwoorden)", n.Name(), id)
				}
			}
			if sc.vraag != "" && (len(sc.goed) == 0 || sc.goed[0] == "") {
				t.Errorf("%s: scène %q heeft een vraag zonder antwoord", n.Name(), id)
			}
			if sc.einde == "" && !invoer && len(sc.keuzes) == 0 {
				t.Errorf("%s: scène %q is een doodlopend eind zonder einde", n.Name(), id)
			}
			if len(sc.keuzes) > 9 {
				t.Errorf("%s: scène %q heeft meer dan 9 keuzes", n.Name(), id)
			}
			if sc.kaart != "" {
				kaart, code, _ := strings.Cut(sc.kaart, " ")
				gevonden := false
				for _, d := range topoKaartVan(kaart).Items {
					gevonden = gevonden || d.Code == strings.TrimSpace(code)
				}
				if !gevonden {
					t.Errorf("%s: scène %q: kaart %q niet gevonden", n.Name(), id, sc.kaart)
				}
			}
			for _, al := range sc.alineas {
				if strings.Contains(al, "—") {
					t.Errorf("%s: scène %q bevat een gedachtestreepje", n.Name(), id)
				}
				if al != "" && strings.ContainsAny(al[:1], "?>=%|#") && !strings.HasPrefix(al, plaatjeTeken) && !strings.HasPrefix(al, tekeningTeken) {
					t.Errorf("%s: scène %q: losse regel die op een code lijkt: %q", n.Name(), id, al)
				}
				if pl, ok := strings.CutPrefix(al, plaatjeTeken); ok {
					rijen := strings.Split(pl, "\n")
					if len(rijen) > 24 {
						t.Errorf("%s: scène %q: plaatje te hoog (hooguit 24 pixels)", n.Name(), id)
					}
					for _, r := range rijen {
						if len([]rune(r)) > 60 {
							t.Errorf("%s: scène %q: plaatje te breed voor de Eee PC", n.Name(), id)
						}
						for _, c := range r {
							if _, ok := plaatjeKleuren[c]; !ok && c != '.' {
								t.Errorf("%s: scène %q: onbekende kleur %q in een plaatje", n.Name(), id, c)
							}
						}
					}
				}
				if tk, ok := strings.CutPrefix(al, tekeningTeken); ok {
					for _, r := range strings.Split(tk, "\n") {
						if len([]rune(r)) > 60 {
							t.Errorf("%s: scène %q: tekening te breed voor de Eee PC", n.Name(), id)
						}
					}
				}
			}
		}
		if len(a.eindes) < 2 {
			t.Errorf("%s: maar %d eindes", n.Name(), len(a.eindes))
		}
		// robotpuzzels: de oplossing in het commentaar moet werken
		oplossingen := map[string]string{}
		for _, m := range oplossingRe.FindAllStringSubmatch(string(data), -1) {
			oplossingen[m[1]] = m[2]
		}
		for id, sc := range a.scenes {
			if sc.robot == "" {
				continue
			}
			op, ok := oplossingen[id]
			if !ok {
				t.Errorf("%s: robotpuzzel %q zonder // oplossing %s: ...", n.Name(), id, id)
				continue
			}
			r, _ := leesRooster(sc.robot)
			if _, gehaald, fout := draaiRobot(r, op); !gehaald {
				t.Errorf("%s: robotpuzzel %q: de oplossing %q werkt niet (%s)", n.Name(), id, op, fout)
			}
		}
	}
}

// Willekeurige toetsen in elk avontuur: niets mag crashen.
func TestAvonturenCrashenNiet(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 29
	toetsen := []string{"1", "2", "3", "4", "9", "0", "enter", "n", "esc", "up", "down", "pgdown", "backspace", "x", "245", "1648", "2,5", "vooruit 2, rechts", "herhaal 3: vooruit 1"}
	for _, a := range alleAvonturen() {
		rng := rand.New(rand.NewPCG(1, 2))
		m = toets(m, tabToets(tabLezen))
		m.ls.plank = 1
		m.openAvontuur("avontuur:" + a.id)
		for i := 0; i < 3000; i++ {
			if m.ls.avontuur == nil {
				m.openAvontuur("avontuur:" + a.id)
			}
			k := toetsen[rng.IntN(len(toetsen))]
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			switch k {
			case "enter":
				msg = tea.KeyMsg{Type: tea.KeyEnter}
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			case "backspace":
				msg = tea.KeyMsg{Type: tea.KeyBackspace}
			case "up":
				msg = tea.KeyMsg{Type: tea.KeyUp}
			case "down":
				msg = tea.KeyMsg{Type: tea.KeyDown}
			case "pgdown":
				msg = tea.KeyMsg{Type: tea.KeyPgDown}
			}
			nm, _ := m.Update(msg) // zonder de opdrachten (knipperende cursor) af te wachten
			m = nm.(model)
			if i%10 == 0 {
				_ = m.View()
			}
		}
		m = toets(m, "esc")
	}
}

func TestRobot(t *testing.T) {
	r, err := leesRooster("5x3 start=0,2,oost doel=4,0 muur=2,2 2,1")
	if err != nil {
		t.Fatal(err)
	}
	for prog, wil := range map[string]bool{
		"vooruit 1, links, vooruit 2, rechts, vooruit 3": true,
		"vooruit 4": false, // botst tegen de muur
		"vooruit 1, links, vooruit 2, rechts, herhaal 3: vooruit 1": true,
	} {
		if _, gehaald, _ := draaiRobot(r, prog); gehaald != wil {
			t.Errorf("%q: gehaald %v, verwacht %v", prog, gehaald, wil)
		}
	}
	if _, _, fout := draaiRobot(r, "spring"); !strings.Contains(fout, "kent") {
		t.Errorf("onbekende opdracht: %q", fout)
	}
	if _, _, fout := draaiRobot(r, "herhaal 999: vooruit 1"); fout == "" {
		t.Error("herhaal 999 hoort geweigerd te worden")
	}
	if !antwoordGoed(" 2,5 ", []string{"2.5"}) || !antwoordGoed("Kwart over DRIE", []string{"kwart over drie"}) || antwoordGoed("", []string{""}) {
		t.Error("antwoorden vergelijken")
	}
}

// De klokkentoren van begin tot het beste einde, zoals een kind hem speelt; voortgang wordt
// bewaard en n n begint opnieuw.
func TestKlokkentorenSpelen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 29
	m = toets(m, tabToets(tabLezen))
	m = toets(m, "tab") // avonturen
	if !strings.Contains(m.View(), "Het geheim van de klokkentoren") {
		t.Fatalf("de plank met avonturen:\n%s", m.View())
	}
	m = toets(m, "enter")
	speel := func(stappen ...string) {
		for _, s := range stappen {
			nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
			m = voer(nm.(model), cmd)
			if s != "1" && s != "2" && s != "3" {
				m = toets(m, "enter")
			}
		}
	}
	speel("1", "1", "2") // Lotte wakker, samen kijken, eerst de lerarenkamer
	speel("2", "1")      // sleutel laten liggen, de trap op
	if m.ls.avontuur.scene != "trap" || !m.ls.avontuur.heeft("zaklamp") {
		t.Fatalf("scène %q, spul %v", m.ls.avontuur.scene, m.ls.avontuur.spul)
	}
	// halverwege stoppen: de plek is bewaard
	m = toets(m, "esc")
	m = toets(m, "enter")
	if m.ls.avontuur.scene != "trap" || !m.ls.avontuur.heeft("zaklamp") {
		t.Fatalf("voortgang niet bewaard: %q %v", m.ls.avontuur.scene, m.ls.avontuur.spul)
	}
	speel("1", "1") // zaklamp aan, slot bekijken (zonder sleutel)
	// fout: je blijft bij de vraag en het maatje helpt, steeds een beetje meer
	speel("312")
	if m.ls.avontuur.scene != "slot2" || !strings.Contains(m.View(), "klopt niet") || !strings.Contains(m.View(), "Bliep:") || m.ls.avontuur.hint != 1 {
		t.Fatalf("fout antwoord: %q\n%s", m.ls.avontuur.scene, m.View())
	}
	speel("312") // hetzelfde nog eens telt niet
	if m.ls.avontuur.hint != 1 {
		t.Fatalf("hetzelfde foute antwoord gaf een nieuwe tip (%d)", m.ls.avontuur.hint)
	}
	speel("?") // zelf om een tip vragen
	if m.ls.avontuur.hint != 2 || !strings.Contains(m.ls.avontuur.tip, "kwart voor drie") {
		t.Fatalf("? hoort de volgende tip te geven: %q", m.ls.avontuur.tip)
	}
	speel("111", "222")
	if !strings.Contains(m.ls.avontuur.tip, "2 _ _") {
		t.Fatalf("na de tips hoort het maatje een stukje voor te zeggen: %q", m.ls.avontuur.tip)
	}
	speel("?") // met ? komen er geen letters bij
	speel("333", "444", "555")
	if !strings.Contains(m.ls.avontuur.tip, "2 4 _") || strings.Contains(m.ls.avontuur.tip, "2 4 5") {
		t.Fatalf("het maatje zegt nooit alles voor: %q", m.ls.avontuur.tip)
	}
	speel("245") // goed
	speel("2", "1", "1648", "1")
	if m.ls.avontuur.scene != "schat" || !strings.Contains(m.View(), "tijdcapsule van 1926") || !strings.Contains(m.View(), "1 van de 4 eindes") {
		t.Fatalf("het beste einde:\n%s", m.View())
	}
	// opnieuw spelen begint bij het begin, zonder spullen
	m = toets(m, "1")
	if m.ls.avontuur.scene != "start" || len(m.ls.avontuur.spul) != 0 {
		t.Errorf("opnieuw spelen: %q %v", m.ls.avontuur.scene, m.ls.avontuur.spul)
	}
	// n n begint ook opnieuw
	speel("1")
	m = toets(m, "n")
	m = toets(m, "n")
	if m.ls.avontuur.scene != "start" {
		t.Errorf("n n hoort opnieuw te beginnen: %q", m.ls.avontuur.scene)
	}
	_ = fmt.Sprint
}

func TestMaskers(t *testing.T) {
	for _, c := range []struct {
		antwoord string
		n        int
		want     string
	}{
		{"1648", 1, "1 _ _ _"}, {"1648", 9, "1 6 4 _"}, {"deur open", 5, "d e u r   o _ _ _"},
		{"n", 3, "_"}, {"1/4", 1, "1 / _"}, {"4:30", 2, "4 : 3 _"},
	} {
		if got := maskerAntwoord(c.antwoord, c.n); got != c.want {
			t.Errorf("maskerAntwoord(%q, %d) = %q, wil %q", c.antwoord, c.n, got, c.want)
		}
	}
	if got := maskerProgramma("vooruit 3, links, vooruit 2", 1); got != "vooruit 3, ..." {
		t.Errorf("programma 1: %q", got)
	}
	if got := maskerProgramma("vooruit 3, links, vooruit 2", 9); got != "vooruit 3, links, ..." {
		t.Errorf("programma 9: %q", got)
	}
}

// Een robotpuzzel en een uitweg met Tab, in een klein proefverhaal.
func TestAvontuurHulpBijRobotEnUitweg(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 100, 30
	a, err := leesAvontuur("proef", `titel: Proef
# start
Een robot.
robot: 4x3 start=0,2,oost doel=3,0 -> klaar
// oplossing start: vooruit 3, links, vooruit 2
tip: eerst naar rechts
tip: dan omhoog
anders: Geef het op -> weg
# klaar
== einde: Gelukt
# weg
== einde: Opgegeven`)
	if err != nil {
		t.Fatal(err)
	}
	m = toets(m, tabToets(tabLezen))
	m = toets(m, "tab")
	m.ls.avontuur = &avStaat{av: a, invoer: textinput.New()}
	m.avNaarScene("start")
	typ := func(s string) {
		for _, r := range s {
			m = toets(m, string(r))
		}
		m = toets(m, "enter")
	}
	typ("vooruit 1")
	typ("vooruit 2")
	typ("links")
	if m.ls.avontuur.scene != "start" || !strings.Contains(m.ls.avontuur.tip, "vooruit 3, ...") {
		t.Fatalf("robot: scène %q, tip %q", m.ls.avontuur.scene, m.ls.avontuur.tip)
	}
	if v := m.View(); !strings.Contains(v, "Tab: Geef het op") {
		t.Fatalf("de uitweg hoort in beeld te staan:\n%s", v)
	}
	m = toets(m, "tab")
	if m.ls.avontuur.scene != "weg" {
		t.Fatalf("Tab hoort de uitweg te nemen: %q", m.ls.avontuur.scene)
	}
}
