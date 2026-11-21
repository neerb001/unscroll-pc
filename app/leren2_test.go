package main

import (
	"strings"
	"testing"
)

func TestTopoKaarten(t *testing.T) {
	t.Setenv("TERM", "foot")
	rood := palet{acc: "#ff0000"}
	for _, o := range topoOefeningen {
		dingen := o.dingen()
		if len(dingen) < 5 {
			t.Errorf("%s: maar %d dingen", o.sleutel, len(dingen))
		}
		h := 18
		b := topoMaat(o.kaart, h)
		heel := topoKaartVan(o.kaart).Kader
		leeg := topoTekst(o.kaart, "", false, false, b, h, rood, "nl", heel)
		for _, d := range dingen {
			if d.NL == "" || (o.vraag == "stad" && (d.Stad == nil || d.Stad.NL == "")) {
				t.Errorf("%s: naam of hoofdstad ontbreekt", d.Code)
				continue
			}
			// op het scherm van de Eee PC moet alles te zien zijn, ook Luxemburg en Vlieland
			if topoTekst(o.kaart, d.Code, false, false, b, h, rood, "nl", heel) == leeg {
				t.Errorf("%s (%s) is niet te zien op de kaart", d.NL, d.Code)
			}
			// ingezoomd op het scherm van de Eee PC: het ding staat in beeld
			z := topoZoom(o.kaart, d, 108, 25)
			if z[2]-z[0] <= 0 || z[3]-z[1] <= 0 {
				t.Fatalf("%s: lege uitsnede %v", d.Code, z)
			}
			zLeeg := topoTekst(o.kaart, "", false, false, 108, 25, rood, "nl", z)
			if topoTekst(o.kaart, d.Code, false, false, 108, 25, rood, "nl", z) == zLeeg {
				t.Errorf("%s (%s) is ingezoomd niet te zien", d.NL, d.Code)
			}
		}
	}
	nl := topoOefening{kaart: "nederland"}.dingen()
	telt := map[string]int{}
	for _, d := range nl {
		telt[d.Groep]++
	}
	if telt["provincie"] != 12 || telt["stad"] < 30 || telt["water"] < 15 {
		t.Errorf("Nederland: %v", telt)
	}
	europa := topoOefeningen[1].dingen()
	vind := func(lijst []*topoDing, code string) *topoDing {
		for _, d := range lijst {
			if d.Code == code {
				return d
			}
		}
		t.Fatalf("%s niet gevonden", code)
		return nil
	}
	for _, g := range []struct{ antwoord, code string }{
		{"oekraine", "UKR"}, {"Oekraïne", "UKR"}, {"wit rusland", "BLR"}, {"Belarus", "BLR"}, {"groot-brittannië", "GBR"},
		{"verenigd koninkrijk", "GBR"}, {"bosnie en herzegovina", "BIH"}, {"Noord Macedonie", "MKD"},
	} {
		d := vind(europa, g.code)
		if !topoGoed(g.antwoord, append([]string{d.NL, d.EN}, d.Ook...)...) {
			t.Errorf("%q hoort goed te zijn voor %s", g.antwoord, g.code)
		}
	}
	bra := vind(nl, "prov:Noord-Brabant")
	if !topoGoed("den bosch", append([]string{bra.Stad.NL}, bra.Stad.Ook...)...) || !topoGoed("'s-Hertogenbosch", bra.Stad.NL) {
		t.Error("Den Bosch hoort goed te zijn als hoofdstad van Noord-Brabant")
	}
	if topoGoed("", "Nederland") || topoGoed("Belgie", "Nederland") {
		t.Error("leeg of fout hoort niet goed te zijn")
	}
}

func TestTopoOverhoren(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 29
	m = toets(m, tabToets(tabLeren))
	for i := 0; i < 5; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // topografie
	if !strings.Contains(m.View(), "Nederland") || !strings.Contains(m.View(), "Provinciehoofdsteden") {
		t.Fatalf("keuzemenu:\n%s", m.View())
	}
	m = toets(m, "enter") // Europa rondkijken
	tp := m.lr.topo
	eerste := topoOefeningen[0].dingen()[0]
	if !strings.Contains(m.View(), eerste.NL) || !strings.Contains(m.View(), eerste.Stad.NL) {
		t.Fatalf("rondkijken:\n%s", m.View())
	}
	m = toets(m, "right")
	if !strings.Contains(m.View(), topoOefeningen[0].dingen()[1].NL) {
		t.Error("→ hoort naar het volgende land te gaan")
	}
	m = toets(m, "esc")
	m = toets(m, "down")
	m = toets(m, "enter") // landen
	if len(tp.ronde) != 10 {
		t.Fatalf("overhoren niet begonnen: %d", len(tp.ronde))
	}
	land := tp.ronde[0]
	if strings.Contains(m.View(), land.NL) {
		t.Fatalf("de naam hoort niet in beeld te staan:\n%s", m.View())
	}
	m = toets(m, land.NL)
	m = toets(m, "enter")
	if !tp.goed || !strings.Contains(m.View(), "Goed!") {
		t.Fatalf("goed antwoord niet goed: %q", tp.uitslag)
	}
	m = toets(m, "enter")
	tweede := tp.ronde[1]
	m = toets(m, "atlantis")
	m = toets(m, "enter")
	if tp.goed || !strings.Contains(m.View(), tweede.NL) || tp.ronde[len(tp.ronde)-1] != tweede {
		t.Errorf("fout antwoord: %q", tp.uitslag)
	}
	if st := laadLeren()[topoOefeningen[1].set()]; st[land.Code].Goed != 1 || st[tweede.Code].Fout != 1 {
		t.Errorf("leermotor: %+v %+v", st[land.Code], st[tweede.Code])
	}
	// Nederland: provinciehoofdsteden
	m = toets(m, "esc")
	for i := 0; i < 4; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter")
	p := tp.ronde[0]
	if tp.oefening().sleutel != "topo.nl.capitals" || !strings.Contains(m.View(), "hoofdstad van "+p.NL) {
		t.Fatalf("provinciehoofdstad:\n%s", m.View())
	}
	m = toets(m, p.Stad.NL)
	m = toets(m, "enter")
	if !tp.goed {
		t.Errorf("%s is de hoofdstad van %s: %q", p.Stad.NL, p.NL, tp.uitslag)
	}
	if n := strings.Count(m.View(), "\n") + 1; n > 29 {
		t.Errorf("past niet op het scherm van de Eee PC: %d regels", n)
	}
	// rivieren en wateren: de vraag past bij het soort ding
	m = toets(m, "esc")
	m = toets(m, "down")
	m = toets(m, "down")
	m = toets(m, "enter")
	d := tp.ronde[0]
	if !strings.Contains(m.View(), m.topoVraag(tp.oefening(), d)) {
		t.Errorf("vraag bij %s:\n%s", d.NL, m.View())
	}
}

func TestEngelseWoordjes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var gezegd []string
	oud := spreekEngels
	spreekEngels = func(s string) bool { gezegd = append(gezegd, s); return true }
	defer func() { spreekEngels = oud }()
	gezien := map[string]bool{}
	for _, th := range engelsThemas {
		if tr("nl", "engels.theme."+th.sleutel) == "engels.theme."+th.sleutel {
			t.Errorf("thema %s heeft geen naam", th.sleutel)
		}
		for _, w := range engelsWoorden(th.sleutel) {
			if w.nl == "" || len(w.en) == 0 || w.en[0] == "" {
				t.Errorf("%s: leeg woord %+v", th.sleutel, w)
			}
			if gezien[w.sleutel()] {
				t.Errorf("%s dubbel", w.sleutel())
			}
			gezien[w.sleutel()] = true
		}
	}
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 29
	m = toets(m, tabToets(tabLeren))
	for i := 0; i < 6; i++ {
		m = toets(m, "down")
	}
	m = toets(m, "enter") // Engels
	m = toets(m, "enter") // dieren, vertalen
	e := m.lr.engels
	w := e.ronde[0]
	if !strings.Contains(m.View(), "Hoe zeg je "+w.nl) {
		t.Fatalf("vraag:\n%s", m.View())
	}
	m = toets(m, strings.ToUpper(w.en[0]))
	m = toets(m, "enter")
	if !e.goed || len(gezegd) != 1 || gezegd[0] != w.en[0] {
		t.Fatalf("goed antwoord (hoofdletters maken niet uit), dan uitspreken: %v %v", e.goed, gezegd)
	}
	m = toets(m, "enter")
	m = toets(m, "esc")
	// luisteren en schrijven: het woord wordt meteen uitgesproken
	m = toets(m, "right")
	m = toets(m, "enter")
	if !e.luisteren || len(gezegd) != 2 || gezegd[1] != e.ronde[0].en[0] || strings.Contains(m.View(), e.ronde[0].nl) {
		t.Errorf("luisteren: %v, %v\n%s", e.luisteren, gezegd, m.View())
	}
	// meer goede antwoorden: kip = chicken of hen
	e.ronde[0] = engelsWoord{nl: "kip", en: []string{"chicken", "hen"}}
	m = toets(m, "hen")
	m = toets(m, "enter")
	if !e.goed {
		t.Error("hen hoort ook goed te zijn")
	}
}

func TestKorteVerhalen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	verhalen := korteVerhalen()
	if len(verhalen) < 10 {
		t.Fatalf("maar %d verhalen", len(verhalen))
	}
	for _, v := range verhalen {
		if v.titel == "" || len(v.alineas) < 8 || len(v.vragen) < 2 || v.woorden < 300 {
			t.Errorf("%s: titel %q, %d alinea's, %d vragen, %d woorden", v.id, v.titel, len(v.alineas), len(v.vragen), v.woorden)
		}
		for _, a := range v.alineas {
			if strings.Contains(a, "—") {
				t.Errorf("%s: gedachtestreepje", v.id)
			}
		}
	}
	m, _ := testModel(t, Geheimen{})
	m.inst.Taal = "nl"
	m.breed, m.hoog = 112, 29
	m = toets(m, tabToets(tabLezen))
	if !strings.Contains(m.View(), verhalen[0].titel) || !strings.Contains(m.View(), "minuten") {
		t.Fatalf("de verhalen staan niet op de eerste plank:\n%s", m.View())
	}
	m = toets(m, "enter")
	if m.ls.boek == nil || m.ls.boek.titel != verhalen[0].titel {
		t.Fatalf("verhaal niet geopend:\n%s", m.View())
	}
	for i := 0; i < 20; i++ {
		m = toets(m, " ")
	}
	if !strings.Contains(m.View(), "Om over na te denken") {
		t.Errorf("de vragen staan niet aan het eind:\n%s", m.View())
	}
	m = toets(m, "esc")
	if !strings.Contains(m.View(), "✓") {
		t.Errorf("een uitgelezen verhaal hoort zo te heten:\n%s", m.View())
	}
	m = toets(m, "tab")
	if m.ls.plank != 1 {
		t.Error("Tab hoort naar de e-books te gaan")
	}
}
