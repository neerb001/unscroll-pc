package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Verder met een verhaal: verhalen staan als tekstbestand in de schrijfhoek. Bij het hervatten
// krijgt het maatje het verhaal tot nu toe mee, zodat het verder kan waar jullie waren.

type verhaalBestand struct {
	pad, titel string
	tijd       time.Time
}

func verhalenMap() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, ".local", "share", "unscroll", "verhalen")
}

// eerdereVerhalen geeft de nieuwste verhalen eerst; de titel is het begin van de eerste regel.
func eerdereVerhalen(max int) []verhaalBestand {
	namen, _ := filepath.Glob(filepath.Join(verhalenMap(), "*.txt"))
	sort.Sort(sort.Reverse(sort.StringSlice(namen)))
	var uit []verhaalBestand
	for _, n := range namen {
		data, err := os.ReadFile(n)
		if err != nil || len(strings.TrimSpace(string(data))) == 0 {
			continue
		}
		titel, _, _ := strings.Cut(strings.TrimSpace(string(data)), "\n")
		if r := []rune(titel); len(r) > 50 {
			titel = string(r[:49]) + "…"
		}
		v := verhaalBestand{pad: n, titel: titel}
		if t, err := time.ParseInLocation("2006-01-02-1504", strings.TrimSuffix(filepath.Base(n), ".txt"), time.Local); err == nil {
			v.tijd = t
		}
		uit = append(uit, v)
		if len(uit) == max {
			break
		}
	}
	return uit
}

// verderMetVerhaal opent een eerder verhaal; dat telt niet als een nieuwe verhaalsessie.
func (m *model) verderMetVerhaal(v verhaalBestand) tea.Cmd {
	mj := m.mj
	if reden := m.maatjeDicht(); reden != "" {
		m.zeg(reden)
		return nil
	}
	data, err := os.ReadFile(v.pad)
	if err != nil {
		m.zeg(m.t("buddy.err"))
		return nil
	}
	tekst := strings.TrimSpace(string(data))
	mj.volgID++
	mj.kiesVerhaal = false
	s := &sessie{id: mj.volgID, modus: modusVerhaal, verhaal: v.pad, geteld: true, eerder: tekst}
	staart := []rune(tekst)
	if len(staart) > 700 {
		staart = append([]rune("… "), staart[len(staart)-700:]...)
	}
	s.regels = []regel{{2, m.t("buddy.sofar")}, {1, string(staart)}, {1, m.t("buddy.continue")}}
	mj.sessie = s
	mj.invoer.SetValue("")
	return mj.invoer.Focus()
}

// metEerder zet het verhaal tot nu toe voor het eerste bericht na het hervatten.
func metEerder(s *sessie, bericht string) string {
	if s.eerder == "" || len(s.gesprek) > 0 {
		return bericht
	}
	return "This is our story so far, written together in an earlier session:\n\n" + s.eerder +
		"\n\n(End of the story so far. We continue now. The child writes:)\n\n" + bericht
}
