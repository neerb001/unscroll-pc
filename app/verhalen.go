package main

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
)

// Korte verhalen voor groep 7 en 8, naast de e-books in Lezen. Een e-book is lang; een kort
// verhaal lees je in vijf minuten uit. Aan het eind staan een paar vragen om over na te denken
// (begrijpend lezen), zonder antwoorden of punten.
//
// De verhalen staan als tekstbestanden in app/verhalen en komen met de app mee:
//
//	Titel
//
//	Alinea's, gescheiden door een lege regel.
//
//	---
//	Een vraag per regel.

//go:embed verhalen/*.txt
var verhalenBestanden embed.FS

type kortVerhaal struct {
	id, titel string
	alineas   []string
	vragen    []string
	woorden   int
}

func korteVerhalen() []kortVerhaal {
	namen, _ := verhalenBestanden.ReadDir("verhalen")
	var uit []kortVerhaal
	for _, n := range namen {
		data, err := verhalenBestanden.ReadFile(path.Join("verhalen", n.Name()))
		if err != nil {
			continue
		}
		tekst, vragen, _ := strings.Cut(strings.TrimSpace(string(data)), "\n---\n")
		blokken := strings.Split(strings.TrimSpace(tekst), "\n\n")
		v := kortVerhaal{id: "verhaal:" + strings.TrimSuffix(n.Name(), ".txt"), titel: strings.TrimSpace(blokken[0])}
		for _, b := range blokken[1:] {
			b = strings.TrimSpace(strings.ReplaceAll(b, "\n", " "))
			if b != "" {
				v.alineas = append(v.alineas, b)
				v.woorden += len(strings.Fields(b))
			}
		}
		for _, r := range strings.Split(vragen, "\n") {
			if r = strings.TrimSpace(r); r != "" {
				v.vragen = append(v.vragen, r)
			}
		}
		uit = append(uit, v)
	}
	sort.Slice(uit, func(i, j int) bool { return uit[i].id < uit[j].id })
	return uit
}

func zoekVerhaal(id string) (kortVerhaal, bool) {
	for _, v := range korteVerhalen() {
		if v.id == id {
			return v, true
		}
	}
	return kortVerhaal{}, false
}

func isVerhaal(id string) bool { return strings.HasPrefix(id, "verhaal:") }

// verhalenLijst: de verhalen als lijst om uit te kiezen, met hoe lang je ongeveer leest
// (een kind in groep 7 leest zo'n 120 woorden per minuut stil).
func (m *model) verhalenLijst() lijst {
	g := laadLeesGeheugen()
	var items []item
	for _, v := range korteVerhalen() {
		items = append(items, item{id: v.id, titel: v.titel, info: m.t("read.minutes", max((v.woorden+60)/120, 1)),
			klaar: g.Uit[v.id], soort: soortEboek})
	}
	return lijst{items: items}
}

// verhaalAlineas: de titel, het verhaal en de vragen, klaar voor de leesweergave.
func (m *model) verhaalAlineas(v kortVerhaal) []alinea {
	uit := []alinea{{tekst: v.titel, kop: true}}
	for _, a := range v.alineas {
		uit = append(uit, alinea{tekst: a})
	}
	if len(v.vragen) > 0 {
		uit = append(uit, alinea{tekst: m.t("read.think"), kop: true})
		for i, q := range v.vragen {
			uit = append(uit, alinea{tekst: fmt.Sprintf("%d. %s", i+1, q)})
		}
	}
	return uit
}
