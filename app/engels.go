package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Engelse woordjes voor groep 7 en 8, per thema. Twee manieren: vertalen (je ziet het
// Nederlandse woord en typt het Engelse) en luisteren (je hoort het Engelse woord en schrijft
// het op). Na elk antwoord hoor je hoe je het uitspreekt. De leermotor laat terugkomen wat
// lastig is.

type engelsThema struct {
	sleutel string
	woorden []string // "nederlands=engels", met / voor meer goede antwoorden
}

var engelsThemas = []engelsThema{
	{"dieren", []string{"hond=dog", "kat=cat", "paard=horse", "koe=cow", "varken=pig", "schaap=sheep", "vogel=bird", "vis=fish", "muis=mouse", "konijn=rabbit", "olifant=elephant", "aap=monkey", "beer=bear", "slang=snake", "kip=chicken/hen", "eend=duck"}},
	{"eten", []string{"brood=bread", "kaas=cheese", "melk=milk", "water=water", "appel=apple", "banaan=banana", "ei=egg", "boter=butter", "suiker=sugar", "soep=soup", "aardappel=potato", "groente=vegetables/vegetable", "fruit=fruit", "ontbijt=breakfast", "lunch=lunch", "avondeten=dinner"}},
	{"school", []string{"school=school", "leraar=teacher", "juf=teacher/miss", "klas=class/classroom", "boek=book", "pen=pen", "potlood=pencil", "gum=eraser/rubber", "schrift=notebook", "tas=bag/schoolbag", "les=lesson", "huiswerk=homework", "pauze=break", "vraag=question", "antwoord=answer", "schoolplein=playground/schoolyard"}},
	{"lichaam", []string{"hoofd=head", "haar=hair", "oog=eye", "oor=ear", "neus=nose", "mond=mouth", "tand=tooth", "hand=hand", "vinger=finger", "arm=arm", "been=leg", "voet=foot", "knie=knee", "buik=belly/stomach", "rug=back", "hart=heart"}},
	{"familie", []string{"moeder=mother/mum/mom", "vader=father/dad", "broer=brother", "zus=sister", "opa=grandfather/grandpa", "oma=grandmother/grandma", "oom=uncle", "tante=aunt", "neef=cousin/nephew", "nicht=cousin/niece", "baby=baby", "kind=child/kid", "familie=family", "vriend=friend", "buurman=neighbour/neighbor"}},
	{"huis", []string{"huis=house/home", "deur=door", "raam=window", "keuken=kitchen", "slaapkamer=bedroom", "badkamer=bathroom", "tuin=garden", "trap=stairs", "tafel=table", "stoel=chair", "bed=bed", "bank=sofa/couch", "dak=roof", "muur=wall", "vloer=floor", "lamp=lamp"}},
	{"kleding", []string{"broek=trousers/pants", "trui=sweater/jumper", "jas=coat/jacket", "schoen=shoe", "sok=sock", "jurk=dress", "rok=skirt", "hoed=hat", "muts=hat/beanie", "sjaal=scarf", "handschoen=glove", "riem=belt", "pyjama=pyjamas/pajamas", "laars=boot", "overhemd=shirt"}},
	{"weer", []string{"weer=weather", "zon=sun", "regen=rain", "sneeuw=snow", "wind=wind", "wolk=cloud", "onweer=thunderstorm/storm", "warm=warm/hot", "koud=cold", "lente=spring", "zomer=summer", "herfst=autumn/fall", "winter=winter", "mist=fog/mist", "regenboog=rainbow"}},
	{"tijd", []string{"maandag=monday", "dinsdag=tuesday", "woensdag=wednesday", "donderdag=thursday", "vrijdag=friday", "zaterdag=saturday", "zondag=sunday", "januari=january", "februari=february", "maart=march", "april=april", "mei=may", "juni=june", "juli=july", "augustus=august", "september=september", "oktober=october", "november=november", "december=december", "vandaag=today", "morgen=tomorrow", "gisteren=yesterday", "week=week", "maand=month", "jaar=year"}},
	{"doen", []string{"lopen=walk", "rennen=run", "zwemmen=swim", "springen=jump", "lezen=read", "schrijven=write", "praten=talk/speak", "luisteren=listen", "kijken=look/watch", "eten=eat", "drinken=drink", "slapen=sleep", "spelen=play", "zingen=sing", "helpen=help", "koken=cook", "kopen=buy", "denken=think", "weten=know", "begrijpen=understand"}},
	{"woorden", []string{"groot=big/large", "klein=small/little", "mooi=beautiful/pretty", "lelijk=ugly", "snel=fast/quick", "langzaam=slow", "blij=happy", "verdrietig=sad", "boos=angry", "moe=tired", "nieuw=new", "oud=old", "goed=good", "slecht=bad", "makkelijk=easy", "moeilijk=difficult/hard"}},
}

type engelsWoord struct {
	nl  string
	en  []string
	set string
}

func (w engelsWoord) sleutel() string { return w.nl + "=" + w.en[0] }

func engelsWoorden(thema string) []engelsWoord {
	var uit []engelsWoord
	for _, t := range engelsThemas {
		if thema != dicteeAlles && t.sleutel != thema {
			continue
		}
		for _, r := range t.woorden {
			nl, en, _ := strings.Cut(r, "=")
			uit = append(uit, engelsWoord{nl: nl, en: strings.Split(en, "/"), set: t.sleutel})
		}
	}
	return uit
}

// spreekEngels: het Engelse woord uitspreken (espeak-ng, Brits Engels zoals op school).
var spreekEngels = func(tekst string) bool { return spreekStem("en-gb", tekst) }

const (
	engelsKiezen = iota
	engelsBezig
	engelsKlaar
)

type engelsStaat struct {
	scherm    int
	keuze     int  // thema, de laatste is "alles door elkaar"
	luisteren bool // luisteren en schrijven in plaats van vertalen
	thema     string
	ronde     []engelsWoord
	nr        int
	invoer    textinput.Model
	nagekeken bool
	goed      bool
	gehad     map[string]bool
	lastig    []string
	geenStem  bool
}

func nieuwEngels() *engelsStaat {
	e := &engelsStaat{}
	e.invoer = textinput.New()
	e.invoer.CharLimit = 30
	e.invoer.Prompt = "✎ "
	return e
}

func engelsSet(luisteren bool, thema string) string {
	if luisteren {
		return "engels:luister:" + thema
	}
	return "engels:" + thema
}

func (m *model) engelsStart() tea.Cmd {
	e := m.lr.engels
	e.thema = dicteeAlles
	if e.keuze < len(engelsThemas) {
		e.thema = engelsThemas[e.keuze].sleutel
	}
	per := map[string]engelsWoord{}
	var sleutels []string
	for _, w := range engelsWoorden(e.thema) {
		per[w.sleutel()] = w
		sleutels = append(sleutels, w.sleutel())
	}
	e.ronde = nil
	for _, s := range laadLeren().kies(engelsSet(e.luisteren, e.thema), sleutels, 10, time.Now()) {
		e.ronde = append(e.ronde, per[s])
	}
	e.nr, e.gehad, e.lastig, e.nagekeken = 0, map[string]bool{}, nil, false
	e.scherm = engelsBezig
	e.invoer.SetValue("")
	if e.luisteren {
		e.geenStem = !spreekEngels(e.ronde[0].en[0])
	}
	return e.invoer.Focus()
}

func (m *model) engelsNakijken() {
	e := m.lr.engels
	w := e.ronde[e.nr]
	antwoord := strings.ToLower(strings.TrimSpace(e.invoer.Value()))
	antwoord = strings.TrimPrefix(strings.TrimPrefix(antwoord, "to "), "the ")
	goed := false
	for _, en := range w.en {
		if antwoord == en {
			goed = true
		}
	}
	if !e.gehad[w.sleutel()] {
		e.gehad[w.sleutel()] = true
		l := laadLeren()
		l.zet(engelsSet(e.luisteren, e.thema), w.sleutel(), goed, time.Now())
		l.bewaar()
	}
	e.goed, e.nagekeken = goed, true
	if !goed {
		e.lastig = append(e.lastig, w.nl+" = "+w.en[0])
		if len(e.ronde) < 14 {
			e.ronde = append(e.ronde, w)
		}
	}
	e.geenStem = !spreekEngels(w.en[0]) // zo spreek je het uit
}

func (m *model) toetsEngels(msg tea.KeyMsg) (tea.Cmd, bool) {
	e := m.lr.engels
	k := msg.String()
	switch e.scherm {
	case engelsKiezen:
		switch k {
		case "up", "k":
			e.keuze = max(e.keuze-1, 0)
		case "down", "j":
			e.keuze = min(e.keuze+1, len(engelsThemas))
		case "left", "right", "tab":
			e.luisteren = !e.luisteren
		case "enter":
			return m.engelsStart(), true
		case "esc":
			m.lr.scherm = lerenMenu
		default:
			return nil, false
		}
		return nil, true
	case engelsKlaar:
		switch k {
		case "enter", " ":
			return m.engelsStart(), true
		case "esc":
			e.scherm = engelsKiezen
		}
		return nil, true
	}
	switch k {
	case "esc":
		e.invoer.Blur()
		e.scherm = engelsKiezen
		return nil, true
	case "tab":
		e.geenStem = !spreekEngels(e.ronde[e.nr].en[0])
		return nil, true
	case "enter":
		switch {
		case e.nagekeken:
			e.nr++
			e.nagekeken = false
			e.invoer.SetValue("")
			if e.nr >= len(e.ronde) {
				e.scherm = engelsKlaar
				e.invoer.Blur()
			} else if e.luisteren {
				e.geenStem = !spreekEngels(e.ronde[e.nr].en[0])
			}
		case strings.TrimSpace(e.invoer.Value()) != "":
			m.engelsNakijken()
		}
		return nil, true
	}
	if e.nagekeken {
		return nil, true
	}
	var cmd tea.Cmd
	e.invoer, cmd = e.invoer.Update(msg)
	return cmd, true
}

func (m model) viewEngels(p palet, kop string, breed int) string {
	e := m.lr.engels
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	switch e.scherm {
	case engelsKiezen:
		l := laadLeren()
		manier := m.t("engels.translate")
		if e.luisteren {
			manier = m.t("engels.listen")
		}
		regels := []string{m.t("engels.mode", acc.Render(manier)), ""}
		for i := 0; i <= len(engelsThemas); i++ {
			thema := dicteeAlles
			if i < len(engelsThemas) {
				thema = engelsThemas[i].sleutel
			}
			var sl []string
			for _, w := range engelsWoorden(thema) {
				sl = append(sl, w.sleutel())
			}
			naam := fmt.Sprintf("%-24s", m.t("engels.theme."+thema))
			stip := dim.Render(fmt.Sprintf("  %d/%d", l.kent(engelsSet(e.luisteren, thema), sl), len(sl)))
			if i == e.keuze {
				regels = append(regels, acc.Render("▸ "+naam)+stip)
			} else {
				regels = append(regels, "  "+naam+stip)
			}
		}
		regels = append(regels, "", dim.Render(m.t("dictee.legend")))
		return kop + "\n\n" + paneel(p, m.t("learn.english"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("engels.pickkeys"))
	case engelsKlaar:
		regels := []string{acc.Render(m.t("dictee.done")), ""}
		if len(e.lastig) == 0 {
			regels = append(regels, m.t("dictee.allright"))
		} else {
			regels = append(regels, m.t("dictee.tricky"))
			for _, x := range e.lastig {
				regels = append(regels, "  "+x)
			}
		}
		return kop + "\n\n" + paneel(p, m.t("learn.english"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(m.t("dictee.donekeys"))
	}
	w := e.ronde[e.nr]
	vraag := m.t("engels.ask", acc.Render(w.nl))
	if e.luisteren {
		vraag = m.t("engels.askheard")
	}
	regels := []string{dim.Render(m.t("dictee.count", e.nr+1, len(e.ronde))), "", "  " + vraag, "", e.invoer.View(), ""}
	if e.geenStem && e.luisteren {
		regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Width(breed-6).Render(m.t("engels.nosound", w.nl)), "")
	}
	if e.nagekeken {
		juist := strings.Join(w.en, " / ")
		if e.goed {
			regels = append(regels, lipgloss.NewStyle().Foreground(p.groen).Render(m.t("engels.right", w.nl, juist)))
		} else {
			regels = append(regels, lipgloss.NewStyle().Foreground(p.geel).Render(m.t("engels.wrong", w.nl, juist)))
		}
	}
	toetsen := m.t("engels.keys")
	if e.nagekeken {
		toetsen = m.t("topo.nextkeys")
	}
	return kop + "\n\n" + paneel(p, m.t("learn.english"), strings.Join(regels, "\n"), breed, true) + "\n" + dim.Render(toetsen)
}
