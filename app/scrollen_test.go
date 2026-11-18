package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVenster(t *testing.T) {
	tekst := "a\nb\nc\nd\ne\nf"
	boven := 0
	if venster("a\nb", &boven, 5, "meer") != "a\nb" {
		t.Error("korte tekst hoort er heel te staan, zonder meer-regel")
	}
	if got := venster(tekst, &boven, 3, "meer"); got != "a\nb\nmeer" {
		t.Errorf("begin: %q", got)
	}
	boven = helemaalOnder
	// helemaal onderaan: geen "er is meer" meer
	if got := venster(tekst, &boven, 3, "meer"); got != "d\ne\nf" || boven != 3 {
		t.Errorf("eind: %q (boven %d)", got, boven)
	}
	// na End moet één keer ↑ meteen iets doen
	scrollToets("up", &boven, 3)
	if got := venster(tekst, &boven, 3, "meer"); got != "c\nd\nmeer" {
		t.Errorf("↑ na het eind: %q", got)
	}
	if scrollToets("x", &boven, 3) {
		t.Error("x is geen scrolltoets")
	}
	scrollToets("home", &boven, 3)
	if boven != 0 {
		t.Error("Home hoort naar het begin te gaan")
	}
	scrollToets("up", &boven, 3)
	if boven != 0 {
		t.Error("boven het begin kan niet")
	}
}

// Een eerder verhaal hervatten: het hele verhaal moet te lezen zijn, niet alleen het eind,
// en de zinnen van het kind horen niet onder de naam van het maatje te staan.
func TestHervatVerhaalHeelTeLezen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _, _ := maatjeModel(t)
	m.hoog = 20
	m = toets(m, "3")
	m = toets(m, "up") // Samen een verhaal
	m = toets(m, "enter")
	m = typ(m, "Er was eens een draak die bang was voor vuur")
	for i := 0; i < 8; i++ {
		m = typ(m, "en toen gebeurde er iets geks met de maan")
	}
	m = toets(m, "esc")
	m = toets(m, "enter")
	m = toets(m, "down") // het eerdere verhaal
	m = toets(m, "enter")
	if m.mj.sessie == nil || m.mj.sessie.verhaal == "" {
		t.Fatalf("verhaal niet hervat:\n%s", m.View())
	}
	if strings.Contains(m.View(), "draak") {
		t.Fatal("in een klein scherm hoort het begin eerst buiten beeld te zijn")
	}
	for i := 0; i < 80; i++ {
		m = toets(m, "up")
	}
	if !strings.Contains(m.View(), "Er was eens een draak") {
		t.Errorf("met ↑ moet het begin van het verhaal te lezen zijn:\n%s", m.View())
	}
	if strings.Contains(m.View(), "Bliep: Er was eens") || strings.Contains(m.View(), "Bliep: …") {
		t.Errorf("de zinnen van het kind staan onder de naam van het maatje:\n%s", m.View())
	}
}

// Ouders zien vier weken aan gesprekken: in een lange lijst blijft de keuze in beeld.
func TestLangeGesprekkenlijstScrollt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _, _ := maatjeModel(t)
	m.hoog = 20
	dir := sessieMap(modusHuiswerk)
	_ = os.MkdirAll(dir, 0o700)
	nu := time.Now()
	for i := 0; i < 40; i++ {
		naam := nu.Add(-time.Duration(i)*time.Hour).Format("2006-01-02-1504") + "-huiswerk.txt"
		_ = os.WriteFile(filepath.Join(dir, naam), []byte(fmt.Sprintf("» vraag nummer %d\n\nantwoord\n", i)), 0o600)
	}
	m.ouderModus = true
	m = toets(m, "3")
	m = toets(m, "h")
	if !strings.Contains(m.View(), "vraag nummer 0") || strings.Contains(m.View(), "vraag nummer 35") {
		t.Fatalf("de lijst hoort bij het begin te staan:\n%s", m.View())
	}
	for i := 0; i < 35; i++ {
		m = toets(m, "down")
	}
	if !strings.Contains(m.View(), "▸ ") || !strings.Contains(m.View(), "vraag nummer 35") {
		t.Errorf("de gekozen sessie is uit beeld:\n%s", m.View())
	}
	if n := strings.Count(m.View(), "\n"); n > m.hoog {
		t.Errorf("de lijst is hoger dan het scherm: %d regels", n)
	}
	// het kind kan ook verder met meer dan zes eerdere opdrachten
	m.ouderModus = false
	m = toets(m, "esc")
	m.mj.kiesVerhaal = false
	m.mj.keuze = modusHuiswerk
	m = toets(m, "enter")
	if len(m.mj.verhalen) <= 6 {
		t.Errorf("het kind ziet maar %d eerdere opdrachten", len(m.mj.verhalen))
	}
}

func TestWeekoverzichtScrollt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _, _ := maatjeModel(t)
	m.hoog = 20
	nu := time.Now()
	for i := 6; i >= 0; i-- {
		for j := 0; j < 4; j++ {
			schrijfLog(logRegel{Tijd: nu.AddDate(0, 0, -i).Add(time.Duration(j) * time.Minute), Modus: modusVraag, Vraag: fmt.Sprintf("vraag %d-%d", i, j), Oordeel: oordeelOK})
		}
	}
	m.ouderModus = true
	m = toets(m, "3")
	m = toets(m, "w")
	if !strings.Contains(m.View(), "vraag 0-3") || strings.Contains(m.View(), "vraag 6-0") {
		t.Fatalf("het overzicht hoort bij de nieuwste te beginnen:\n%s", m.View())
	}
	m = toets(m, "home")
	if !strings.Contains(m.View(), "vraag 6-0") {
		t.Errorf("ouders kunnen niet terug naar eerdere dagen:\n%s", m.View())
	}
}

func TestNatuurdagboekOudeNotitiesTeLezen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.hoog = 20
	for i := 0; i < 15; i++ {
		bewaarNotitie(notitie{time.Now().Add(time.Duration(i) * time.Minute), fmt.Sprintf("notitie nummer %d", i)})
	}
	m = toets(m, "4")
	m = toets(m, "down")
	m = toets(m, "down")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "notitie nummer 14") || strings.Contains(m.View(), "notitie nummer 0") {
		t.Fatalf("de nieuwste hoort bovenaan te staan:\n%s", m.View())
	}
	m = toets(m, "end")
	if !strings.Contains(m.View(), "notitie nummer 0") {
		t.Errorf("de oudste notitie is niet te bereiken:\n%s", m.View())
	}
	m = toets(m, "n") // schrijven werkt nog gewoon
	if !m.od.schrijven {
		t.Error("n hoort een nieuwe notitie te beginnen")
	}
}

func TestDierEnWoordenboekScrollen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.hoog = 20
	var lang []string
	for i := 0; i < 30; i++ {
		lang = append(lang, fmt.Sprintf("Zin %d over de das.", i))
	}
	m.tab, m.od.scherm = tabOntdekken, ontdekDier
	m.od.dier = &dierInfo{Titel: "Das", Omschrijving: "zoogdier", Tekst: strings.Join(lang, "\n")}
	if strings.Contains(m.View(), "Zin 29") {
		t.Fatal("de lange tekst hoort niet in één keer te passen")
	}
	m = toets(m, "end")
	if !strings.Contains(m.View(), "Zin 29") {
		t.Errorf("het eind van de diertekst is niet te bereiken:\n%s", m.View())
	}

	m.tab, m.sh.scherm = tabSchrijfhoek, schrijfWoord
	m.sh.wb.uitleg = &woordUitleg{Woord: "das", Betekenissen: lang[:8]}
	m.sh.wb.invoer.SetValue("das")
	m.hoog = 14
	if strings.Contains(m.View(), "Zin 7") {
		t.Fatal("de uitleg hoort niet in één keer te passen")
	}
	for i := 0; i < 20; i++ { // End is hier voor het tekstveld
		m = toets(m, "pgdown")
	}
	if !strings.Contains(m.View(), "Zin 7") || !strings.Contains(m.View(), "das") {
		t.Errorf("de uitleg scrollt niet, of het tekstveld is weg:\n%s", m.View())
	}
}
