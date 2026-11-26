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
	{"v0.38.0", [2]string{
		"Missie Bliep: nu staat er elke dag een baas klaar! Kon je een dag niet? Dan heb je de volgende dag twee gevechten. Er zijn vier nieuwe bazen, en na het eerste schip gaat het verhaal verder: op de maan, op Mars en bij Bliep thuis.",
		"Mission Bliep: now there's a boss waiting every day! Missed a day? Then you get two fights the next day. There are four new bosses, and after the first ship the story goes on: on the Moon, on Mars and at Bliep's home.",
	}},
	{"v0.37.0", [2]string{
		"Vast bij een vraag in een avontuur? Bliep helpt je nu: na een fout antwoord krijg je een tip, en met ? vraag je er zelf om. Eerst een duwtje, dan hoe je het kunt bedenken. Het antwoord typ je altijd zelf.",
		"Stuck on a question in an adventure? Bliep helps you now: after a wrong answer you get a hint, and ? asks for one. First a nudge, then how to work it out. You always type the answer yourself.",
	}},
	{"v0.36.0", [2]string{
		"Nieuw bovenaan 3 Leren: Mijn heelal (al je vakken als sterren die feller gaan schijnen) en Missie Bliep: met goede antwoorden repareer je het ruimteschip van Bliep, en elke week wacht er een baas. Bij Bliep kun je nu ook zelf de leraar zijn: kies Leg het uit.",
		"New at the top of 3 Learn: My universe (all your subjects as stars that shine brighter) and Mission Bliep: right answers repair Bliep's spaceship, and every week a boss is waiting. With Bliep you can now be the teacher too: choose Explain it.",
	}},
	{"v0.35.0", [2]string{
		"Nieuw bij 2 Lezen: elf avonturen waarin jij kiest wat er gebeurt. Los sommen, codes en kaartpuzzels op, stuur een robot en zoek alle eindes. Druk op Tab bij Lezen.",
		"New in 2 Read: eleven adventures where you choose what happens. Solve sums, codes and map puzzles, steer a robot and find every ending. Press Tab in Read.",
	}},
	{"v0.34.0", [2]string{
		"Je kunt de computer nu zelf uitzetten of opnieuw opstarten: ga naar 9 Systeem en druk twee keer op z (uitzetten) of r (opnieuw opstarten).",
		"You can now shut down or restart the computer yourself: go to 9 System and press z (shut down) or r (restart) twice.",
	}},
	{"v0.32.0", [2]string{
		"De kaarten bij Topografie zijn nu ook duidelijk als de computer in de eenvoudige tekststand draait: blauwe zee, zwart land en het gevraagde land in het geel.",
		"The maps in Geography are now clear in the simple text mode too: blue sea, black land and the asked country in yellow.",
	}},
	{"v0.31.0", [2]string{
		"De kaarten bij Topografie gebruiken nu het hele scherm en zoomen in op wat gevraagd wordt. Met Tab zie je de hele kaart.",
		"The maps in Geography now use the whole screen and zoom in on what is asked. Tab shows the whole map.",
	}},
	{"v0.29.0", [2]string{
		"De kaarten bij Topografie zijn nu veel duidelijker: blauwe zee, witte grenzen en bij rondkijken de namen op de kaart. En bij de tafels: ga met de pijltjes naar een tafel en druk op Enter.",
		"The maps in Geography are much clearer now: blue sea, white borders and the names on the map when you look around. And in the times tables: go to a table with the arrows and press Enter.",
	}},
	{"v0.28.0", [2]string{
		"Nieuw bij 3 Leren: Geschiedenis, met een tijdlijn van de tien tijdvakken. En bij Topografie kun je nu ook de hele wereld oefenen.",
		"New in 3 Learn: History, with a timeline of the ten eras. And in Geography you can now practise the whole world too.",
	}},
	{"v0.27.0", [2]string{
		"Nieuw bij 2 Lezen: tien korte verhalen, elk zo'n vijf minuten lezen. Met Tab ga je naar je e-books. En bij 3 Leren: topografie van Europa en Nederland, en Engelse woordjes.",
		"New in 2 Read: ten short stories, about five minutes each. Tab takes you to your e-books. And in 3 Learn: geography of Europe and the Netherlands, and English words.",
	}},
	{"v0.26.0", [2]string{
		"Nieuw: 3 Leren, met dictee, flitslezen en klokkijken. De tafels en de rekenmachine staan daar nu ook. De knoppen daarna schuiven een plekje op: je maatje is nu 4, Denksport 6 en Systeem 9.",
		"New: 3 Learn, with dictation, flash reading and telling time. The times tables and the calculator are there now too. The buttons after it move up one place: the buddy is now 4, Puzzles 6 and System 9.",
	}},
	{"v0.25.3", [2]string{
		"In het computerlab staat nu duidelijker wat je kunt typen: typ missie en druk op Enter om de opdracht nog eens te zien. Typ je opnieuw, dan vraagt het lab eerst of je echt alles opnieuw wilt.",
		"The computer lab now shows more clearly what you can type: type mission and press Enter to see the mission again. If you type restart, the lab first asks whether you really want to start everything over.",
	}},
	{"v0.25.2", [2]string{
		"Programmeren werkt weer: je Hedy-programma's draaien nu meteen, vanaf de eerste opdracht. Ga naar Ontdekken en kies Programmeren.",
		"Programming works again: your Hedy programs now run straight away, from the very first exercise. Go to Discover and choose Programming.",
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
		"Het schaakbord is veel groter, met stukken die je goed herkent. Naast het bord staat welk plaatje de koning, dame, toren, loper, paard en pion is. Ga naar Denksport en kies Schaken.",
		"The chess board is much bigger, with pieces that are easy to recognise. Next to the board you can see which picture is the king, queen, rook, bishop, knight and pawn. Go to Puzzles and choose Chess.",
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
