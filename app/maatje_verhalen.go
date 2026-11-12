package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Verhalen en huiswerksessies worden bewaard als tekstbestand, zodat het kind later verder kan
// waar het was. Verhalen staan in de schrijfhoek en zijn privé; huiswerk kunnen ouders teruglezen
// in de ouderstand (h). Bij het hervatten krijgt het maatje het gesprek tot nu toe mee.

type verhaalBestand struct {
	pad, titel string
	tijd       time.Time
	modus      int
}

const kindTeken = "» " // regels van het kind in een huiswerkbestand

func unscrollData(sub string) string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, ".local", "share", "unscroll", sub)
}

func verhalenMap() string { return unscrollData("verhalen") }

func sessieMap(modus int) string {
	if modus == modusVerhaal {
		return verhalenMap()
	}
	return unscrollData("huiswerk")
}

var modusBestand = map[int]string{modusHuiswerk: "-huiswerk", modusOverhoren: "-overhoren", modusVraag: "-vraag"}

// bewaarDagen: zo lang blijven gesprekken (niet de verhalen) en de logboeken bewaard.
const bewaarDagen = 28

// opruimen haalt gesprekken en logregels weg die ouder zijn dan vier weken. Verhalen blijven:
// die zijn van het kind.
func opruimen(nu time.Time) {
	grens := nu.AddDate(0, 0, -bewaarDagen)
	namen, _ := filepath.Glob(filepath.Join(sessieMap(modusHuiswerk), "*.txt"))
	for _, n := range namen {
		if t, err := time.ParseInLocation("2006-01-02", filepath.Base(n)[:min(10, len(filepath.Base(n)))], time.Local); err == nil && t.Before(grens) {
			_ = os.Remove(n)
		}
	}
	for _, pad := range []string{logPad(), spLogPad()} {
		data, err := os.ReadFile(pad)
		if err != nil {
			continue
		}
		var houden []string
		for _, r := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var x struct {
				Tijd time.Time `json:"tijd"`
			}
			if json.Unmarshal([]byte(r), &x) == nil && x.Tijd.After(grens) {
				houden = append(houden, r)
			}
		}
		if len(houden) < len(strings.Split(strings.TrimSpace(string(data)), "\n")) {
			_ = os.WriteFile(pad, []byte(strings.Join(houden, "\n")+"\n"), 0o644)
		}
	}
}

// eerdereSessies geeft de nieuwste eerst. Modus -1: alle huiswerk- en overhoorsessies (voor ouders).
func eerdereSessies(modus, max int) []verhaalBestand {
	zoek := modus
	if modus < 0 {
		zoek = modusHuiswerk
	}
	namen, _ := filepath.Glob(filepath.Join(sessieMap(zoek), "*.txt"))
	sort.Sort(sort.Reverse(sort.StringSlice(namen)))
	var uit []verhaalBestand
	for _, n := range namen {
		basis := strings.TrimSuffix(filepath.Base(n), ".txt")
		bm := modusVerhaal
		for md, achter := range modusBestand {
			if strings.HasSuffix(basis, achter) {
				bm, basis = md, strings.TrimSuffix(basis, achter)
			}
		}
		if (modus >= 0 && bm != modus) || (modus < 0 && bm == modusVerhaal) {
			continue
		}
		data, err := os.ReadFile(n)
		if err != nil || len(strings.TrimSpace(string(data))) == 0 {
			continue
		}
		titel, _, _ := strings.Cut(strings.TrimSpace(string(data)), "\n")
		titel = strings.TrimPrefix(titel, kindTeken)
		if r := []rune(titel); len(r) > 50 {
			titel = string(r[:49]) + "…"
		}
		v := verhaalBestand{pad: n, titel: titel, modus: bm}
		if t, err := time.ParseInLocation("2006-01-02-1504", basis, time.Local); err == nil {
			v.tijd = t
		}
		uit = append(uit, v)
		if len(uit) == max {
			break
		}
	}
	return uit
}

// eerdereVerhalen blijft voor de verhalen.
func eerdereVerhalen(max int) []verhaalBestand { return eerdereSessies(modusVerhaal, max) }

// leesGesprek maakt van een huiswerkbestand weer regels: blokken met » zijn van het kind.
func leesGesprek(tekst string) []regel {
	var uit []regel
	for _, blok := range strings.Split(strings.TrimSpace(tekst), "\n\n") {
		blok = strings.TrimSpace(blok)
		if blok == "" {
			continue
		}
		if strings.HasPrefix(blok, kindTeken) {
			uit = append(uit, regel{0, strings.TrimPrefix(blok, kindTeken)})
		} else if n := len(uit); n > 0 && uit[n-1].wie == 1 {
			uit[n-1].tekst += "\n\n" + blok // een antwoord met meerdere alinea's
		} else {
			uit = append(uit, regel{1, blok})
		}
	}
	return uit
}

// verderMet opent een eerdere sessie. Verder gaan telt niet als een nieuwe sessie.
// alleenLezen is voor ouders: teruglezen zonder iets te sturen.
func (m *model) verderMet(v verhaalBestand, alleenLezen bool) tea.Cmd {
	mj := m.mj
	if !alleenLezen {
		if reden := m.maatjeDicht(); reden != "" {
			m.zeg(reden)
			return nil
		}
	}
	data, err := os.ReadFile(v.pad)
	if err != nil {
		m.zeg(m.t("buddy.err"))
		return nil
	}
	tekst := strings.TrimSpace(string(data))
	mj.volgID++
	mj.kiesVerhaal = false
	s := &sessie{id: mj.volgID, modus: v.modus, verhaal: v.pad, geteld: true, eerder: tekst, alleenLezen: alleenLezen}
	if v.modus == modusVerhaal {
		staart := []rune(tekst)
		if len(staart) > 700 {
			staart = append([]rune("… "), staart[len(staart)-700:]...)
		}
		s.regels = []regel{{2, m.t("buddy.sofar")}, {1, string(staart)}, {1, m.t("buddy.continue")}}
	} else {
		s.regels = append(leesGesprek(tekst), regel{2, m.t("buddy.hwback")})
		if alleenLezen {
			s.regels = leesGesprek(tekst)
		}
	}
	mj.sessie = s
	mj.invoer.SetValue("")
	if alleenLezen {
		mj.invoer.Blur()
		return nil
	}
	return mj.invoer.Focus()
}

func (m *model) verderMetVerhaal(v verhaalBestand) tea.Cmd { return m.verderMet(v, false) }

// metEerder zet het gesprek tot nu toe voor het eerste bericht na het hervatten.
func metEerder(s *sessie, bericht string) string {
	if s.eerder == "" || len(s.gesprek) > 0 {
		return bericht
	}
	if s.modus == modusVerhaal {
		return "This is our story so far, written together in an earlier session:\n\n" + s.eerder +
			"\n\n(End of the story so far. We continue now. The child writes:)\n\n" + bericht
	}
	return "This is the earlier part of this session (blocks starting with » are from the child, the others are your answers):\n\n" +
		s.eerder + "\n\n(End of the earlier part. Carry on where we left off. The child writes:)\n\n" + bericht
}

// bewaarSessie schrijft een beurt weg: verhalen in de schrijfhoek, huiswerk in de huiswerkmap.
func (m *model) bewaarSessie(s *sessie, kind, maatje string) {
	if s.verhaal == "" {
		dir := sessieMap(s.modus)
		if dir == "" {
			return
		}
		_ = os.MkdirAll(dir, 0o700)
		s.verhaal = filepath.Join(dir, time.Now().Format("2006-01-02-1504")+modusBestand[s.modus]+".txt")
	}
	f, err := os.OpenFile(s.verhaal, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	if s.modus == modusVerhaal {
		fmt.Fprintf(f, "%s\n\n%s\n\n", kind, maatje)
	} else {
		fmt.Fprintf(f, "%s%s\n\n%s\n\n", kindTeken, kind, maatje)
	}
}
