package main

import "strings"

// Lange tekst op een klein scherm (een Eee PC heeft maar zo'n 30 regels): overal dezelfde toetsen.
// ↑ ↓ een regel, PgUp PgDn een scherm, Home End naar het begin of het eind.
// boven is de eerste zichtbare regel; venster begrenst hem, zodat de toetsen dat niet hoeven.

const helemaalOnder = 1 << 30

// scrollToets verschuift boven bij een scrolltoets en zegt of de toets gebruikt is.
func scrollToets(k string, boven *int, pagina int) bool {
	stap := max(pagina-1, 1)
	switch k {
	case "up":
		*boven--
	case "down":
		*boven++
	case "pgup":
		*boven -= stap
	case "pgdown":
		*boven += stap
	case "home":
		*boven = 0
	case "end":
		*boven = helemaalOnder
	default:
		return false
	}
	*boven = max(*boven, 0)
	return true
}

// venster laat hoogte regels zien vanaf boven. Komt er onder nog meer, dan zegt de onderste
// regel dat (meer, al opgemaakt); helemaal onderaan is die regel gewoon tekst.
func venster(tekst string, boven *int, hoogte int, meer string) string {
	regels := strings.Split(tekst, "\n")
	hoogte = max(hoogte, 2)
	if len(regels) <= hoogte {
		*boven = 0
		return tekst
	}
	if *boven >= len(regels)-hoogte {
		*boven = len(regels) - hoogte
		return strings.Join(regels[*boven:], "\n")
	}
	*boven = max(*boven, 0)
	return strings.Join(regels[*boven:*boven+hoogte-1], "\n") + "\n" + meer
}

// lijstVenster: welk stuk van een lijst met n regels te zien is, zodat de gekozen regel
// in beeld blijft. Geeft begin en eind (exclusief).
func lijstVenster(n, gekozen, hoogte int) (int, int) {
	if n <= hoogte {
		return 0, n
	}
	begin := min(max(gekozen-hoogte/2, 0), n-hoogte)
	return begin, begin + hoogte
}
