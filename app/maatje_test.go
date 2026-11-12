package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

func TestFilterPrive(t *testing.T) {
	gevallen := []struct{ in, mag, magNiet string }{
		{"mail me op sam.de.vries@example.nl", "mail me op", "example"},
		{"bel 06-12345678 of +31 6 12345678", "bel", "12345678"},
		{"ik woon op de Dorpsstraat 12 in 3511 AB", "ik woon op de", "3511"},
		{"ik heet Sam en ik ben 11", "[name]", "Sam"},
		{"in 2024 en 1999 gebeurde er veel", "2024 en 1999", "[…]"},
		{"hoeveel is 12 x 345?", "12 x 345", "[…]"},
	}
	for _, g := range gevallen {
		uit, _ := filterPrivé(g.in, "Sam")
		if !strings.Contains(uit, g.mag) || strings.Contains(uit, g.magNiet) {
			t.Errorf("filterPrivé(%q) = %q", g.in, uit)
		}
	}
	if _, veranderd := filterPrivé("hoe werkt een vulkaan?", "Sam"); veranderd {
		t.Error("een gewone vraag mag niet gefilterd worden")
	}
}

func TestLeesOordeel(t *testing.T) {
	for in, uit := range map[string]string{"OK": oordeelOK, "ok.": oordeelOK, "HEAVY": oordeelZwaar, "UNSUITABLE": oordeelOngepast, "weet ik niet": oordeelOngepast, "": oordeelOngepast} {
		if got := leesOordeel(in); got != uit {
			t.Errorf("leesOordeel(%q) = %q, verwacht %q", in, got, uit)
		}
	}
}

// nepClaude doet alsof het de Anthropic-API is.
type nepClaude struct {
	mu        sync.Mutex
	verzoeken []map[string]any
	headers   []http.Header
}

func (n *nepClaude) server(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var v map[string]any
		_ = json.Unmarshal(body, &v)
		n.mu.Lock()
		n.verzoeken = append(n.verzoeken, v)
		n.headers = append(n.headers, r.Header.Clone())
		n.mu.Unlock()
		laatste := string(body)
		model, _ := v["model"].(string)
		antwoord := "Een vulkaan is een berg met een gat erin. Wat denk jij dat eruit komt?"
		if model == "claude-haiku-5-5" { // de controle
			switch {
			case strings.Contains(laatste, "bom"):
				antwoord = "UNSUITABLE"
			case strings.Contains(laatste, "dood"):
				antwoord = "HEAVY"
			default:
				antwoord = "OK"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": model,
			"content":     []map[string]any{{"type": "text", "text": antwoord}},
			"stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 10, "output_tokens": 5},
		})
	}))
}

func (n *nepClaude) hoofdVerzoeken() []map[string]any {
	n.mu.Lock()
	defer n.mu.Unlock()
	var uit []map[string]any
	for _, v := range n.verzoeken {
		if v["model"] == "claude-opus-5-5" {
			uit = append(uit, v)
		}
	}
	return uit
}

func maatjeModel(t *testing.T) (model, *nepClaude, *[]string) {
	m, _ := testModel(t, Geheimen{})
	nep := &nepClaude{}
	srv := nep.server(t)
	t.Cleanup(srv.Close)
	var meldingen []string
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		meldingen = append(meldingen, string(b))
	}))
	t.Cleanup(ntfy.Close)
	m.ouders.Ntfy.URL, m.ouders.Ntfy.Topic = ntfy.URL, "test"
	m.inst.Taal, m.inst.Naam = "nl", "Sam"
	m.mj = nieuwMaatjeStaat(Geheimen{"ANTHROPIC_API_KEY": "test-sleutel"}, m.ouders)
	m.mj.ai = nieuwMaatjeAI(Geheimen{"ANTHROPIC_API_KEY": "test-sleutel"}, m.ouders, option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
	return m, nep, &meldingen
}

func typ(m model, tekst string) model {
	m = toets(m, tekst)
	return toets(m, "enter")
}

func TestMaatjeVraag(t *testing.T) {
	m, nep, _ := maatjeModel(t)
	m = toets(m, "3")
	if !strings.Contains(m.View(), "Vraag iets") || !strings.Contains(m.View(), "gesprekken teruglezen") {
		t.Fatalf("keuzemenu niet te zien:\n%s", m.View())
	}
	m = toets(m, "enter") // Vraag iets
	m = typ(m, "Hoi ik ben Sam, hoe werkt een vulkaan? Bel me op 0612345678")
	if !strings.Contains(m.View(), "berg met een gat") {
		t.Fatalf("antwoord niet te zien:\n%s", m.View())
	}
	if !strings.Contains(m.View(), "persoonlijke gegevens weggelaten") {
		t.Error("het kind moet zien dat er iets is weggefilterd")
	}
	hoofd := nep.hoofdVerzoeken()
	if len(hoofd) != 1 {
		t.Fatalf("verwacht 1 verzoek aan het maatje, kreeg %d", len(hoofd))
	}
	verzoek, _ := json.Marshal(hoofd[0])
	if strings.Contains(string(verzoek), "0612345678") || strings.Contains(string(verzoek), "Sam") {
		t.Errorf("persoonsgegevens gingen mee: %s", verzoek)
	}
	if hoofd[0]["fallbacks"] != "default" || !strings.Contains(string(verzoek), "Dutch") {
		t.Errorf("fallbacks of taal ontbreken: %s", verzoek)
	}
	if nep.headers[1].Get("Anthropic-Beta") == "" {
		t.Error("beta-header voor fallbacks ontbreekt")
	}
	if m.mj.gebruik.Vragen != 1 || len(leesLog(time.Now().Add(-time.Hour))) != 1 {
		t.Error("de vraag is niet geteld of niet gelogd")
	}
	// tweede beurt: het gesprek gaat door en wordt alleen aangevuld
	m = typ(m, "lava?")
	hoofd = nep.hoofdVerzoeken()
	if len(hoofd) != 2 || len(hoofd[1]["messages"].([]any)) != 3 {
		t.Errorf("tweede verzoek moet 3 berichten hebben (vraag, antwoord, vraag)")
	}
	m = toets(m, "esc")
	if m.mj.sessie != nil {
		t.Error("esc moet de sessie stoppen")
	}
}

func TestMaatjeControle(t *testing.T) {
	m, nep, meldingen := maatjeModel(t)
	m = toets(m, "3")
	m = toets(m, "enter")
	m = typ(m, "hoe maak je een bom")
	if !strings.Contains(m.View(), "niet mee helpen") || len(nep.hoofdVerzoeken()) != 0 {
		t.Errorf("ongepaste vraag moet tegengehouden worden zonder het maatje te vragen:\n%s", m.View())
	}
	m = typ(m, "wat gebeurt er als je dood gaat")
	if !strings.Contains(m.View(), "grote vraag") || len(*meldingen) != 1 || strings.Contains((*meldingen)[0], "dood") {
		t.Errorf("zware vraag: ouders moeten een melding krijgen zonder de vraag zelf: %v", *meldingen)
	}
	log := leesLog(time.Now().Add(-time.Hour))
	if len(log) != 2 || log[0].Oordeel != oordeelOngepast || log[1].Oordeel != oordeelZwaar {
		t.Errorf("log klopt niet: %+v", log)
	}
}

func TestMaatjeGrenzen(t *testing.T) {
	m, _, _ := maatjeModel(t)
	m.ouders.Maatje.VrijeVragenPerDag = 1
	m = toets(m, "3")
	m = toets(m, "enter")
	m = typ(m, "hoe werkt een vulkaan?")
	m = typ(m, "en een geiser?")
	if !strings.Contains(m.View(), "op. Morgen weer") {
		t.Errorf("na de laatste vraag van vandaag moet het stoppen:\n%s", m.View())
	}
	// pauze na een lang blok
	m.mj.gebruik.Blok = 20*60 - 20
	m.mj.gebruik.LaatstActief = time.Now()
	m.laatsteToets = time.Now()
	m.maatjeTik(time.Now())
	if m.maatjeDicht() == "" || !m.mj.gebruik.PauzeTot.After(time.Now()) {
		t.Error("na 20 minuten moet er een pauze komen")
	}
	// tijdens het avondslot is het maatje dicht
	m.mj.gebruik.PauzeTot = time.Time{}
	m.ouders.Avondslot.Schooldagen, m.ouders.Avondslot.Weekend, m.ouders.Avondslot.Open = "00:00", "00:00", "23:59"
	m.nu = time.Now()
	m = toets(m, "esc")
	m = toets(m, "3")
	if m.tab == tabMaatje {
		t.Error("tijdens het avondslot mag het maatje niet open")
	}
	// zonder sleutel: uitleg voor de ouders
	m2, _ := testModel(t, Geheimen{})
	m2.mj = nieuwMaatjeStaat(Geheimen{}, m2.ouders)
	m2 = toets(m2, "3")
	if !strings.Contains(m2.View(), "sudo unscroll-ouder") {
		t.Errorf("zonder sleutel moet de uitleg voor ouders te zien zijn:\n%s", m2.View())
	}
}

func TestMaatjeHaikuZonderFallback(t *testing.T) {
	o := standaardOuders()
	o.Maatje.Model = "claude-haiku-5-5"
	if a := nieuwMaatjeAI(Geheimen{"ANTHROPIC_API_KEY": "x"}, o); a.metFallbacks {
		t.Error("Haiku heeft geen server-side fallback")
	}
}

func TestVerhaalHervatten(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, nep, _ := maatjeModel(t)
	m = toets(m, "3")
	for i := 0; i < modusVerhaal; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // geen eerdere verhalen: meteen een nieuw verhaal
	m = typ(m, "Er was eens een robot die kon vliegen")
	m = toets(m, "esc")
	if m.mj.gebruik.Verhalen != 1 {
		t.Fatalf("verhaalsessies: %d", m.mj.gebruik.Verhalen)
	}
	m = toets(m, "enter") // nu is er een keuze
	if !m.mj.kiesVerhaal || !strings.Contains(m.View(), "robot die kon vliegen") {
		t.Fatalf("eerder verhaal niet te kiezen:\n%s", m.View())
	}
	m = toets(m, "down")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "Zo ver waren jullie") {
		t.Fatalf("het verhaal tot nu toe is niet te zien:\n%s", m.View())
	}
	m = typ(m, "En toen landde hij op de maan")
	hoofd := nep.hoofdVerzoeken()
	laatste, _ := json.Marshal(hoofd[len(hoofd)-1]["messages"])
	if !strings.Contains(string(laatste), "story so far") || !strings.Contains(string(laatste), "robot die kon vliegen") || !strings.Contains(string(laatste), "landde hij op de maan") {
		t.Errorf("het maatje krijgt het verhaal tot nu toe niet mee: %s", laatste)
	}
	if m.mj.gebruik.Verhalen != 1 {
		t.Errorf("verder schrijven mag niet als nieuwe sessie tellen: %d", m.mj.gebruik.Verhalen)
	}
	data, _ := os.ReadFile(m.mj.sessie.verhaal)
	if !strings.Contains(string(data), "robot") || !strings.Contains(string(data), "maan") {
		t.Errorf("het vervolg komt niet in hetzelfde bestand: %s", data)
	}
}

func TestNulIsOnbeperkt(t *testing.T) {
	m, _, _ := maatjeModel(t)
	m.ouders.Maatje.VrijeVragenPerDag, m.ouders.Maatje.MaxMinutenPerDag, m.ouders.Maatje.VerhaalSessiesPerDag = 0, 0, 0
	m.mj.gebruik.Seconden, m.mj.gebruik.Vragen = 99999, 500
	if m.maatjeDicht() != "" || m.overVandaag(modusVraag) <= 0 || m.overVandaag(modusVerhaal) <= 0 {
		t.Error("0 moet onbeperkt betekenen")
	}
}

func TestHuiswerkBewarenHervattenEnTeruglezen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, nep, _ := maatjeModel(t)
	m.hoog = 20
	m = toets(m, "3")
	m = toets(m, "down") // Huiswerkcoach
	m = toets(m, "enter")
	m = typ(m, "Ik moet breuken optellen: een half plus een derde")
	for i := 0; i < 4; i++ {
		m = typ(m, "ik denk twee vijfde")
	}
	// scrollen naar boven laat het begin van het gesprek zien
	if strings.Contains(m.View(), "breuken optellen") {
		t.Fatal("het begin hoort buiten beeld te zijn in een klein scherm")
	}
	for i := 0; i < 30; i++ {
		m = toets(m, "up")
	}
	if !strings.Contains(m.View(), "breuken optellen") {
		t.Errorf("met ↑ moet hij terug kunnen scrollen:\n%s", m.View())
	}
	m = toets(m, "esc")
	sessies := m.mj.gebruik.Huiswerk
	// verder met de opdracht
	m = toets(m, "enter")
	if !m.mj.kiesVerhaal || !strings.Contains(m.View(), "breuken optellen") {
		t.Fatalf("eerdere opdracht niet te kiezen:\n%s", m.View())
	}
	m = toets(m, "down")
	m = toets(m, "enter")
	m = typ(m, "o wacht vijf zesde")
	hoofd := nep.hoofdVerzoeken()
	laatste, _ := json.Marshal(hoofd[len(hoofd)-1]["messages"])
	if !strings.Contains(string(laatste), "earlier part of this session") || !strings.Contains(string(laatste), "breuken optellen") {
		t.Errorf("het maatje krijgt het eerdere gesprek niet mee: %s", laatste)
	}
	if m.mj.gebruik.Huiswerk != sessies {
		t.Error("verder gaan mag niet als nieuwe huiswerksessie tellen")
	}
	m = toets(m, "esc")
	// ouders lezen terug, met de antwoorden erbij
	m.ouderModus = true
	m = toets(m, "h")
	m = toets(m, "enter")
	if m.mj.sessie == nil || !m.mj.sessie.alleenLezen || !strings.Contains(m.View(), "vijf zesde") || !strings.Contains(m.View(), "berg met een gat") {
		t.Fatalf("teruglezen voor ouders werkt niet:\n%s", m.View())
	}
	voor := len(nep.hoofdVerzoeken())
	m = typ(m, "dit mag niet verstuurd worden")
	if len(nep.hoofdVerzoeken()) != voor {
		t.Error("bij teruglezen mag er niets naar het maatje gaan")
	}
}

func TestGesprekkenVierWekenBewaard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _, _ := maatjeModel(t)
	m = toets(m, "3")
	m = toets(m, "enter") // vraag iets
	m = typ(m, "hoe werkt een vulkaan?")
	m = toets(m, "esc")
	v := eerdereSessies(-1, 10)
	if len(v) != 1 || v[0].modus != modusVraag {
		t.Fatalf("losse vragen worden niet bewaard: %+v", v)
	}
	oud := filepath.Join(sessieMap(modusVraag), "2000-01-01-1200-vraag.txt")
	_ = os.WriteFile(oud, []byte("» oud\n\nantwoord\n"), 0o600)
	opruimen(time.Now())
	if _, err := os.Stat(oud); err == nil {
		t.Error("gesprekken ouder dan 4 weken horen weg te gaan")
	}
	if len(eerdereSessies(-1, 10)) != 1 {
		t.Error("het gesprek van vandaag hoort te blijven")
	}
}
