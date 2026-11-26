package main

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
)

// De verhalen van Missie Bliep, één bestand per seizoen in app/missies. Een nieuw seizoen
// toevoegen is een nieuw bestand schrijven; de app hoeft er verder niets voor te weten.
//
//	titel: Gestrand op de maan
//	titel.en: Stranded on the Moon
//	intro: Wat er gebeurt aan het begin van het seizoen.
//	0: Het verhaal als onderdeel 0 (de romp) gerepareerd is.
//	...
//	7: Het slot: het schip vliegt weg.
//
// Elke regel mag een Engelse versie hebben met .en erachter. Zijn alle seizoenen gespeeld, dan
// gaan de verhalen opnieuw rond, vanaf seizoen 2 (seizoen 1 is de neerstorting in de tuin).
// Ouders zien in de ouderstand hoe lang de verhalen nog meegaan.

//go:embed missies/*.txt
var missieBestanden embed.FS

type seizoenVerhaal struct {
	tekst map[string]string // sleutel (titel, intro, 0..7, met .en) -> tekst
}

func alleSeizoenen() []seizoenVerhaal {
	namen, _ := missieBestanden.ReadDir("missies")
	sort.Slice(namen, func(i, j int) bool { return namen[i].Name() < namen[j].Name() })
	var uit []seizoenVerhaal
	for _, n := range namen {
		data, err := missieBestanden.ReadFile(path.Join("missies", n.Name()))
		if err != nil {
			continue
		}
		uit = append(uit, leesSeizoen(string(data)))
	}
	return uit
}

func leesSeizoen(tekst string) seizoenVerhaal {
	s := seizoenVerhaal{tekst: map[string]string{}}
	for _, r := range strings.Split(tekst, "\n") {
		r = strings.TrimSpace(r)
		if r == "" || strings.HasPrefix(r, "//") {
			continue
		}
		if k, v, ok := strings.Cut(r, ":"); ok {
			s.tekst[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return s
}

// seizoenVoor: het verhaal bij seizoen n (vanaf 1). Na het laatste geschreven seizoen gaan de
// verhalen opnieuw rond, zonder de neerstorting van seizoen 1.
func seizoenVoor(n int) seizoenVerhaal {
	alle := alleSeizoenen()
	switch {
	case len(alle) == 0:
		return seizoenVerhaal{tekst: map[string]string{}}
	case n <= len(alle):
		return alle[max(n, 1)-1]
	case len(alle) == 1:
		return alle[0]
	}
	return alle[1+(n-len(alle)-1)%(len(alle)-1)]
}

// zeg: de tekst in de taal van het kind (Engels als die er is), anders Nederlands.
func (s seizoenVerhaal) zeg(sleutel, taal string) string {
	if taal == "en" {
		if t := s.tekst[sleutel+".en"]; t != "" {
			return t
		}
	}
	return s.tekst[sleutel]
}

func (s seizoenVerhaal) deel(i int, taal string) string { return s.zeg(fmt.Sprint(i), taal) }
