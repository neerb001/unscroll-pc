package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTafelsOefenen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.dk = nieuweDenksport()
	m.inst.Taal = "nl"
	m = toets(m, "5")
	m = toets(m, "enter") // Tafels oefenen
	m = toets(m, "a")     // alles uit
	m = toets(m, "7")     // alleen de tafel van 7
	m = toets(m, "enter")
	if len(m.dk.vragen) != 10 {
		t.Fatalf("ronde van tien sommen, kreeg %d", len(m.dk.vragen))
	}
	for _, s := range m.dk.vragen {
		if s.a != 7 && s.b != 7 {
			t.Fatalf("som buiten de tafel van 7: %+v", s)
		}
	}
	// eerste fout, de rest goed
	eerste := m.dk.vragen[0]
	m = toets(m, fmt.Sprint(eerste.uitkomst()+1))
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "Bijna") || len(m.dk.vragen) != 11 {
		t.Fatalf("een fout hoort later terug te komen:\n%s", m.View())
	}
	for !m.dk.klaar {
		m = toets(m, fmt.Sprint(m.dk.vragen[m.dk.nr].uitkomst()))
		m = toets(m, "enter")
	}
	if !strings.Contains(m.View(), "Klaar met deze ronde") || !strings.Contains(m.View(), eerste.tekst()) {
		t.Fatalf("einde van de ronde:\n%s", m.View())
	}
	if laadTafels().Reeks[eerste.sleutel()] != 1 {
		t.Error("na fout en daarna goed hoort de reeks op 1 te staan")
	}
	m = toets(m, "o")
	if !strings.Contains(m.View(), "zit erin") {
		t.Errorf("overzicht:\n%s", m.View())
	}
}

func TestRekenmachineScherm(t *testing.T) {
	m, _ := testModel(t, Geheimen{})
	m.dk = nieuweDenksport()
	m.inst.Taal = "nl"
	m = toets(m, "5")
	m = toets(m, "down")
	m = toets(m, "enter")
	m = toets(m, "1/3 + 1/6")
	m = toets(m, "enter")
	m = toets(m, "5/0")
	m = toets(m, "enter")
	v := m.View()
	if !strings.Contains(v, "1/2  = 0,5") || !strings.Contains(v, "delen door nul kan niet") {
		t.Errorf("rekenmachine:\n%s", v)
	}
	m = toets(m, "up")
	if m.dk.reken.Value() != "5/0" {
		t.Error("↑ hoort de vorige som terug te halen")
	}
}

func TestWoordenboek(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Get("action") == "parse" && q.Get("page") == "fiets":
			_, _ = w.Write([]byte(`{"parse":{"title":"fiets","wikitext":"{{=nld=}}\n{{-pron-}}\n*{{sound}}\n{{-noun-|nld}}\n{{nl-noun|fiets|fietsen}}\n'''fiets''' {{f}}\n#{{verkeer|nld}} [[tweewielig]] [[vervoermiddel]] dat je [[voortbewegen|voortbeweegt]] met je [[voet]]en<ref>bron</ref>\n{{bijv-1|Ik ga met de fiets naar school.}}\n#{{vulg|nld}} iets onaardigs\n{{-syn-}}\n*[[rijwiel]], [[stalen ros]]\n{{=eng=}}\n{{-noun-|eng}}\n# something English"}}`))
		case q.Get("action") == "parse":
			_, _ = w.Write([]byte(`{"error":{"code":"missingtitle"}}`))
		case q.Get("action") == "opensearch":
			_, _ = w.Write([]byte(`["fiest",["fiets","fietsen","fiest"]]`))
		case strings.HasSuffix(r.URL.Path, "/definition/bike"):
			_, _ = w.Write([]byte(`{"en":[{"partOfSpeech":"Noun","definitions":[{"definition":"A <a href=\"x\">bicycle</a>."},{"definition":"(vulgar) something rude"}]}]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	oud := wiktBasis
	wiktBasis = srv.URL + "/%s"
	defer func() { wiktBasis = oud }()

	u, err := zoekWoord("nl", "fiets")
	if err != nil || len(u.Betekenissen) != 1 || !strings.Contains(u.Betekenissen[0], "tweewielig vervoermiddel dat je voortbeweegt met je voeten") ||
		u.Soorten[0] != "zelfstandig naamwoord" || strings.Join(u.Synoniemen, ",") != "rijwiel,stalen ros" {
		t.Fatalf("fiets: %+v %v", u, err)
	}
	u, _ = zoekWoord("nl", "fiest")
	if len(u.Betekenissen) != 0 || len(u.Suggesties) != 2 || u.Suggesties[0] != "fiets" {
		t.Fatalf("spelling: %+v", u)
	}
	u, _ = zoekWoord("en", "bike")
	if len(u.Betekenissen) != 1 || u.Betekenissen[0] != "A bicycle." {
		t.Fatalf("bike: %+v", u)
	}
	// via het scherm
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m = toets(m, "6")
	m = toets(m, "down")
	m = toets(m, "enter")
	m = toets(m, "fiest")
	m = toets(m, "enter")
	if !strings.Contains(m.View(), "Bedoel je: fiets, fietsen?") {
		t.Errorf("woordenboekscherm:\n%s", m.View())
	}
}
