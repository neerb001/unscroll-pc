package main

import "fmt"

// Vertalingen. Een nieuwe taal toevoegen:
//  1. voeg de code toe aan talen en taalNamen
//  2. kopieer het blok "en" in teksten en vertaal de waarden
//  3. go test controleert of er geen sleutels ontbreken
var talen = []string{"nl", "en"}

var taalNamen = map[string]string{
	"nl": "Nederlands",
	"en": "English",
}

// standaardTaal: Engels, tenzij er bij Systeem een andere taal is gekozen.
func standaardTaal() string { return "en" }

// tr geeft de vertaling van sleutel in taal. Ontbreekt die, dan Engels, dan de sleutel zelf.
func tr(taal, sleutel string, args ...any) string {
	s, ok := teksten[taal][sleutel]
	if !ok {
		if s, ok = teksten["en"][sleutel]; !ok {
			return sleutel
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

var teksten = map[string]map[string]string{
	"nl": {
		"tab.start": "start", "tab.listen": "luister", "tab.read": "lees", "tab.discover": "ontdek", "tab.puzzles": "denk",
		"tab.write": "schrijf", "tab.today": "vandaag", "tab.system": "systeem",

		"menu.listen": "Luisteren", "menu.listen.d": "muziek, Spotify, luisterboeken, podcasts, radio",
		"menu.read": "Lezen", "menu.read.d": "e-boeken uit je eigen bibliotheek",
		"menu.buddy": "Vraag %s", "menu.buddy.d": "je slimme maatje",
		"menu.discover": "Ontdekken", "menu.discover.d": "computerlab, programmeren, dieren, natuur",
		"menu.puzzles": "Denksport", "menu.puzzles.d": "sudoku, schaakpuzzel, raadsel",
		"menu.write": "Schrijfhoek", "menu.write.d": "privé dagboek en verhalen",
		"menu.today": "Vandaag", "menu.today.d": "weer, wekker, maan, seizoen",
		"menu.system": "Systeem", "menu.system.d": "naam, kleuren, taal, wifi, volume, hulp",

		"greet.night": "Goedenacht", "greet.morning": "Goedemorgen", "greet.afternoon": "Goedemiddag", "greet.evening": "Goedenavond",
		"start.subtitle": "rustige luister- en ontdekcomputer · %s",
		"start.hint":     "Kies met ↑ ↓ en Enter, of druk op een cijfer.",
		"start.today":    "vandaag", "start.now": "nu", "start.buddy": "maatje",
		"start.lock":     "avondstand om %s\nweer open om %s",
		"start.noparent": "ouders.yaml niet gevonden, standaardtijden",
		"start.listen":   "luisteren komt in fase 1",

		"lock.title": "Het is %s. Tijd om tot rust te komen.",
		"lock.sub":   "Ga slapen of lees nog even. De rest is weer open om %s.",
		"lock.read":  "[2] verder lezen", "lock.listen": "[1] luisterboek met slaaptimer", "lock.help": "[8] hulp vragen",
		"lock.night": "Welterusten!", "lock.only": "Avondstand: alleen lezen en je luisterboek.", "lock.status": "avondstand",

		"soon.in":       "komt in %s",
		"soon.building": "Dit onderdeel wordt nog gebouwd. Het komt vanzelf binnen met een update.",
		"soon.listen":   "fase 1\nNavidrome: door artiesten scrollen, typen om te zoeken\nRadio met draaiknop langs goedgekeurde zenders\nLuisterboeken uit Audiobookshelf, voortgang gaat mee\nPodcasts die jullie als ouders kiezen\nSlaaptimer en maximaal volume\nSpotify (fase 3): alleen goedgekeurde albums en lijsten, geen autoplay",
		"soon.read":     "fase 2\nE-boeken uit Audiobookshelf, ook offline\nRustige leesweergave met bladwijzers\nLetters groter en kleiner",
		"soon.buddy":    "fase 4\nVragen in gewone taal via de Anthropic-API\nModi: hoe werkt het, huiswerkcoach, overhoren, samen een verhaal, dieren, mythen\nGrenzen: dagbudget, niet tijdens het avondslot, persoonsgegevens worden weggefilterd\nWeekoverzicht voor de ouders",
		"soon.discover": "fase 5\nComputerlab: Linux leren met missies in een oefenomgeving\nPython, met alleen je eigen kamerlamp als echte uitvoer\nDier van de dag (Wikipedia) en natuurdagboek",
		"soon.puzzles":  "fase 5\nSudoku, schaakpuzzels en raadsels\nGeen timer, geen score, een paar per dag",
		"soon.write":    "fase 5\nPrivé dagboek en verhalen\nAlleen jij kunt het lezen",
		"soon.today":    "fase 2\nWeer met regen per uur en een praktische tip\nWekker, planning, maanfase en ISS\nSeizoen en het volgende jaarfeest",

		"sys.own": "maak hem van jou", "sys.device": "apparaat",
		"sys.computer": "naam computer", "sys.name": "jouw naam", "sys.buddy": "naam maatje", "sys.face": "gezicht", "sys.colors": "kleuren", "sys.lang": "taal",
		"sys.ph.computer": "naam van je computer", "sys.ph.name": "zodat ik je kan begroeten", "sys.ph.buddy": "naam van je maatje",
		"sys.keys":  "↑ ↓ kiezen · typen om te wijzigen · ← → bij gezicht, kleuren en taal · esc klaar",
		"sys.saved": "wijzigingen worden meteen bewaard",
		"sys.wifi":  "wifi", "sys.battery": "accu", "sys.volume": "volume", "sys.version": "versie", "sys.autoupdate": " · bijwerken gaat vanzelf",
		"sys.actions": "[+] [-] volume · [h] hulp vragen aan je ouders", "sys.parent": "[o] ouderinstellingen",
		"sys.pin": "pincode: ", "sys.unknown": "onbekend", "sys.maxvol": " · max %d%%",
		"sys.parentmode": "ouderstand",
		"sys.p.lock":     "avondslot  school %s · weekend %s · open %s",
		"sys.p.after":    "na het slot  lezen %s · luisterboek %s",
		"sys.p.edit":     "wijzigen in /etc/unscroll/ouders.yaml",
		"sys.p.keys":     "[t] avondslot 30 min uit · [Q] app afsluiten · [o] ouderstand uit",
		"yes":            "ja", "no": "nee",

		"msg.saveerr": "Bewaren lukte niet: %s", "msg.parenton": "Ouderstand aan.", "msg.parentoff": "Ouderstand uit.",
		"msg.nopin": "Er is nog geen pincode ingesteld in ouders.yaml.", "msg.badpin": "Pincode klopt niet.",
		"msg.unlock": "Avondslot 30 minuten uit.", "msg.maxvol": "Maximaal volume is %d%%.", "msg.sending": "Berichtje versturen…",
		"help.notset": "Hulp vragen is nog niet ingesteld (ntfy in ouders.yaml).", "help.someone": "Je kind",
		"help.body": "%s vraagt om hulp bij Unscroll.", "help.title": "Hulp gevraagd",
		"help.offline": "Versturen lukte niet. Is de wifi aan?", "help.failed": "Versturen lukte niet (%s).", "help.sent": "Berichtje verstuurd. Er komt zo iemand helpen.",
		"wifi.unknown": "onbekend", "wifi.off": "niet verbonden", "battery.none": "geen accu gevonden", "battery.charging": "%s%% · aan de lader",

		"theme.nacht": "Nachtlucht", "theme.ruimte": "Ruimtestation", "theme.tovenaar": "Tovenaarsschool", "theme.bos": "Bos", "theme.oceaan": "Oceaan",
	},
	"en": {
		"tab.start": "start", "tab.listen": "listen", "tab.read": "read", "tab.discover": "discover", "tab.puzzles": "puzzles",
		"tab.write": "write", "tab.today": "today", "tab.system": "system",

		"menu.listen": "Listen", "menu.listen.d": "music, Spotify, audiobooks, podcasts, radio",
		"menu.read": "Read", "menu.read.d": "e-books from your own library",
		"menu.buddy": "Ask %s", "menu.buddy.d": "your clever buddy",
		"menu.discover": "Discover", "menu.discover.d": "computer lab, coding, animals, nature",
		"menu.puzzles": "Puzzles", "menu.puzzles.d": "sudoku, chess puzzle, riddle",
		"menu.write": "Writing nook", "menu.write.d": "private diary and stories",
		"menu.today": "Today", "menu.today.d": "weather, alarm, moon, season",
		"menu.system": "System", "menu.system.d": "name, colours, language, wifi, volume, help",

		"greet.night": "Good night", "greet.morning": "Good morning", "greet.afternoon": "Good afternoon", "greet.evening": "Good evening",
		"start.subtitle": "calm listening and discovery computer · %s",
		"start.hint":     "Choose with ↑ ↓ and Enter, or press a number.",
		"start.today":    "today", "start.now": "now", "start.buddy": "buddy",
		"start.lock":     "quiet time at %s\nopen again at %s",
		"start.noparent": "ouders.yaml not found, using default times",
		"start.listen":   "listening arrives in phase 1",

		"lock.title": "It's %s. Time to wind down.",
		"lock.sub":   "Go to sleep or read for a bit. Everything else opens again at %s.",
		"lock.read":  "[2] keep reading", "lock.listen": "[1] audiobook with sleep timer", "lock.help": "[8] ask for help",
		"lock.night": "Sleep well!", "lock.only": "Quiet time: only reading and your audiobook.", "lock.status": "quiet time",

		"soon.in":       "coming in %s",
		"soon.building": "This part is still being built. It will arrive by itself with an update.",
		"soon.listen":   "phase 1\nNavidrome: scroll through artists, type to search\nRadio dial with approved stations\nAudiobooks from Audiobookshelf, progress is kept\nPodcasts chosen by parents\nSleep timer and maximum volume\nSpotify (phase 3): approved albums and playlists only, no autoplay",
		"soon.read":     "phase 2\nE-books from Audiobookshelf, also offline\nCalm reading view with bookmarks\nLarger and smaller text",
		"soon.buddy":    "phase 4\nQuestions in plain language via the Anthropic API\nModes: how does it work, homework coach, quiz me, write a story together, animals, myths\nLimits: daily budget, closed during quiet time, personal details are filtered out\nWeekly summary for parents",
		"soon.discover": "phase 5\nComputer lab: learn Linux through missions in a safe sandbox\nPython, with only your own bedroom lamp as real output\nAnimal of the day (Wikipedia) and nature journal",
		"soon.puzzles":  "phase 5\nSudoku, chess puzzles and riddles\nNo timer, no score, a few per day",
		"soon.write":    "phase 5\nPrivate diary and stories\nOnly you can read them",
		"soon.today":    "phase 2\nWeather with rain per hour and a practical tip\nAlarm, schedule, moon phase and ISS passes\nSeason and the next seasonal festival",

		"sys.own": "make it yours", "sys.device": "device",
		"sys.computer": "computer name", "sys.name": "your name", "sys.buddy": "buddy name", "sys.face": "face", "sys.colors": "colours", "sys.lang": "language",
		"sys.ph.computer": "name of your computer", "sys.ph.name": "so I can greet you", "sys.ph.buddy": "name of your buddy",
		"sys.keys":  "↑ ↓ choose · type to change · ← → for face, colours and language · esc done",
		"sys.saved": "changes are saved right away",
		"sys.wifi":  "wifi", "sys.battery": "battery", "sys.volume": "volume", "sys.version": "version", "sys.autoupdate": " · updates install by themselves",
		"sys.actions": "[+] [-] volume · [h] ask your parents for help", "sys.parent": "[o] parent settings",
		"sys.pin": "PIN: ", "sys.unknown": "unknown", "sys.maxvol": " · max %d%%",
		"sys.parentmode": "parent mode",
		"sys.p.lock":     "quiet time  school %s · weekend %s · open %s",
		"sys.p.after":    "after lock  reading %s · audiobook %s",
		"sys.p.edit":     "change in /etc/unscroll/ouders.yaml",
		"sys.p.keys":     "[t] pause quiet time 30 min · [Q] quit app · [o] leave parent mode",
		"yes":            "yes", "no": "no",

		"msg.saveerr": "Could not save: %s", "msg.parenton": "Parent mode on.", "msg.parentoff": "Parent mode off.",
		"msg.nopin": "No PIN has been set in ouders.yaml yet.", "msg.badpin": "Wrong PIN.",
		"msg.unlock": "Quiet time paused for 30 minutes.", "msg.maxvol": "Maximum volume is %d%%.", "msg.sending": "Sending message…",
		"help.notset": "Asking for help is not set up yet (ntfy in ouders.yaml).", "help.someone": "Your child",
		"help.body": "%s is asking for help with Unscroll.", "help.title": "Help requested",
		"help.offline": "Could not send. Is the wifi on?", "help.failed": "Could not send (%s).", "help.sent": "Message sent. Someone will come and help.",
		"wifi.unknown": "unknown", "wifi.off": "not connected", "battery.none": "no battery found", "battery.charging": "%s%% · charging",

		"theme.nacht": "Night sky", "theme.ruimte": "Space station", "theme.tovenaar": "Wizard school", "theme.bos": "Forest", "theme.oceaan": "Ocean",
	},
}
