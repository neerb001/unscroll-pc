package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// "Wat is er nieuw?" Na een update (die vaak 's nachts vanzelf gebeurt) ziet het kind op het
// startscherm een kaartje met wat er veranderd is. Met x gaat het weg; daarna komt het pas terug
// bij een volgende versie met nieuws. Alleen dingen die het kind zelf merkt staan hier.

type nieuwsItem struct {
	versie string
	tekst  [2]string // nl, en
}

// nieuwsLijst: nieuwste bovenaan.
var nieuwsLijst = []nieuwsItem{
	{"v0.25.2", [2]string{
		"Programmeren werkt weer: je Hedy-programma's draaien nu meteen, vanaf de eerste opdracht. Ga naar 4 Ontdekken en kies Programmeren.",
		"Programming works again: your Hedy programs now run straight away, from the very first exercise. Go to 4 Discover and choose Programming.",
	}},
	{"v0.25.1", [2]string{
		"Ga je verder met een eerder verhaal, dan kun je met ↑ en ↓ het hele verhaal teruglezen, vanaf het begin. Ook in je natuurdagboek, bij het dier van de dag en in het woordenboek kun je nu scrollen.",
		"When you carry on with an earlier story, you can read the whole story again with ↑ and ↓, right from the start. You can now also scroll in your nature journal, the animal of the day and the dictionary.",
	}},
	{"v0.25.0", [2]string{
		"Schaken is veel sneller geworden en de stukken zijn mooier getekend. Naast het bord zie je welk stuk wat is.",
		"Chess is much faster now and the pieces are drawn more nicely. Next to the board you can see which piece is which.",
	}},
	{"v0.24.0", [2]string{
		"Zoeken in je boeken: druk bij 2 Lezen op / en typ een stukje van de titel of de schrijver.",
		"Search your books: in 2 Read, press / and type part of the title or the author.",
	}},
	{"v0.22.0", [2]string{
		"Het schaakbord is veel groter, met stukken die je goed herkent. Naast het bord staat welk plaatje de koning, dame, toren, loper, paard en pion is. Ga naar 5 Denksport en kies Schaken.",
		"The chess board is much bigger, with pieces that are easy to recognise. Next to the board you can see which picture is the king, queen, rook, bishop, knight and pawn. Go to 5 Puzzles and choose Chess.",
	}},
}

// versieGetallen: "v0.22.0" wordt [0 22 0]. Iets onleesbaars wordt [0 0 0].
func versieGetallen(v string) [3]int {
	var g [3]int
	fmt.Sscanf(strings.TrimPrefix(v, "v"), "%d.%d.%d", &g[0], &g[1], &g[2])
	return g
}

func versieNieuwer(a, b string) bool {
	ga, gb := versieGetallen(a), versieGetallen(b)
	for i := range ga {
		if ga[i] != gb[i] {
			return ga[i] > gb[i]
		}
	}
	return false
}

// nieuwsSinds: alles wat nieuwer is dan gezien, en niet nieuwer dan de versie die draait.
// Hooguit drie stukjes, zodat het kaartje klein blijft.
func nieuwsSinds(gezien, draait string) []nieuwsItem {
	var uit []nieuwsItem
	for _, n := range nieuwsLijst {
		if versieNieuwer(n.versie, gezien) && !versieNieuwer(n.versie, draait) {
			uit = append(uit, n)
		}
		if len(uit) == 3 {
			break
		}
	}
	return uit
}

// nieuwsBijStart bepaalt bij het opstarten welk nieuws er te laten zien is.
// Een computer die net is ingericht heeft geen nieuws: alles is nieuw. Een computer die al
// instellingen had maar nog nooit nieuws zag (van voor deze functie), krijgt het nieuwste stukje.
func (m *model) nieuwsBijStart(hadInstellingen bool) {
	if versie == "dev" {
		return
	}
	if m.inst.NieuwsGezien == "" {
		if !hadInstellingen {
			m.inst.NieuwsGezien = versie
			_ = m.inst.bewaar()
			return
		}
		if n := nieuwsSinds("", versie); len(n) > 0 {
			m.nieuws = n[:1]
		}
		return
	}
	m.nieuws = nieuwsSinds(m.inst.NieuwsGezien, versie)
}

// nieuwsGelezen: het kaartje weg, en onthouden tot welke versie het kind het nieuws zag.
func (m *model) nieuwsGelezen() {
	if len(m.nieuws) == 0 {
		return
	}
	m.inst.NieuwsGezien = versie
	if versie == "dev" {
		m.inst.NieuwsGezien = m.nieuws[0].versie
	}
	m.nieuws = nil
	m.bewaar()
}

func (m model) nieuwsPaneel(p palet, breed int) string {
	ti := m.taalIndex()
	var r []string
	for i, n := range m.nieuws {
		if i > 0 {
			r = append(r, "")
		}
		r = append(r, lipgloss.NewStyle().Width(breed-4).Render(n.tekst[ti]))
	}
	r = append(r, "", lipgloss.NewStyle().Foreground(p.dim).Render(m.t("news.close")))
	return paneel(p, m.t("news.title"), strings.Join(r, "\n"), breed, true)
}

func bestaatBestand(pad string) bool {
	_, err := os.Stat(pad)
	return err == nil
}
