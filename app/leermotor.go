package main

import (
	"encoding/json"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// De leermotor: één rustige manier van oefenen voor alle onderdelen van Leren (dictee,
// flitslezen, en later topografie en jaartallen). Zonder punten, klok of reeksen.
//
// Elke kaart (een woord, een land, een jaartal) zit in een bakje, zoals bij een Leitner-systeem:
// goed geantwoord gaat hij een bakje verder en komt hij pas na meer dagen terug; fout gaat hij
// terug naar het eerste bakje en komt hij snel weer. Zo oefen je vooral wat nog lastig is.

// leerDagen: na hoeveel dagen een kaart in bakje 0..5 weer aan de beurt is.
var leerDagen = []int{0, 1, 2, 4, 8, 16}

// kentBakje: vanaf dit bakje "ken je hem goed" (voor het overzicht).
const kentBakje = 3

type kaartStand struct {
	Bakje    int    `json:"bakje"`
	Volgende string `json:"volgende"` // datum, 2006-01-02
	Goed     int    `json:"goed"`
	Fout     int    `json:"fout"`
}

// leerStand: per onderdeel (set) per kaart (sleutel) hoe het ervoor staat.
type leerStand map[string]map[string]*kaartStand

func leerPad() string { return filepath.Join(filepath.Dir(instellingenPad()), "leren.json") }

func laadLeren() leerStand {
	l := leerStand{}
	if data, err := os.ReadFile(leerPad()); err == nil {
		_ = json.Unmarshal(data, &l)
	}
	return l
}

func (l leerStand) bewaar() {
	_ = os.MkdirAll(filepath.Dir(leerPad()), 0o755)
	data, _ := json.Marshal(l)
	_ = veiligSchrijven(leerPad(), data, 0o644)
}

func (l leerStand) stand(set, sleutel string) *kaartStand {
	if l[set] == nil {
		l[set] = map[string]*kaartStand{}
	}
	if l[set][sleutel] == nil {
		l[set][sleutel] = &kaartStand{}
	}
	return l[set][sleutel]
}

// kies geeft n kaarten om te oefenen: eerst wat aan de beurt is (laagste bakje eerst), dan
// nieuwe kaarten, en is dat te weinig, dan de rest (wat het langst geleden goed ging eerst).
func (l leerStand) kies(set string, sleutels []string, n int, nu time.Time) []string {
	vandaag := nu.Format("2006-01-02")
	type kandidaat struct {
		sleutel string
		groep   int // 0 aan de beurt, 1 nieuw, 2 nog niet aan de beurt
		bakje   int
		volg    string
		lot     int
	}
	var k []kandidaat
	for _, s := range sleutels {
		c := kandidaat{sleutel: s, lot: rand.IntN(1000)}
		st := l[set][s]
		switch {
		case st == nil:
			c.groep = 1
		case st.Volgende <= vandaag:
			c.groep, c.bakje = 0, st.Bakje
		default:
			c.groep, c.bakje, c.volg = 2, st.Bakje, st.Volgende
		}
		k = append(k, c)
	}
	sort.Slice(k, func(i, j int) bool {
		a, b := k[i], k[j]
		if a.groep != b.groep {
			return a.groep < b.groep
		}
		if a.bakje != b.bakje {
			return a.bakje < b.bakje
		}
		if a.volg != b.volg {
			return a.volg < b.volg
		}
		return a.lot < b.lot
	})
	var uit []string
	for _, c := range k[:min(n, len(k))] {
		uit = append(uit, c.sleutel)
	}
	rand.Shuffle(len(uit), func(i, j int) { uit[i], uit[j] = uit[j], uit[i] })
	return uit
}

// zet onthoudt een antwoord. Alleen het eerste antwoord op een kaart telt in een ronde; dat
// regelt de aanroeper.
func (l leerStand) zet(set, sleutel string, goed bool, nu time.Time) {
	st := l.stand(set, sleutel)
	if goed {
		missieGoed() // energie voor Missie Bliep
		st.Goed++
		st.Bakje = min(st.Bakje+1, len(leerDagen)-1)
	} else {
		st.Fout++
		st.Bakje = 0
	}
	st.Volgende = nu.AddDate(0, 0, leerDagen[st.Bakje]).Format("2006-01-02")
}

// kent: hoeveel van deze kaarten je al goed kent.
func (l leerStand) kent(set string, sleutels []string) int {
	n := 0
	for _, s := range sleutels {
		if st := l[set][s]; st != nil && st.Bakje >= kentBakje {
			n++
		}
	}
	return n
}
