# Unscroll PC

**A calm computer for curious kids. No feeds, no algorithms.**

Unscroll verandert een oude laptop, netbook of Raspberry Pi in een rustige luister- en
ontdekcomputer voor kinderen: muziek, luisterboeken, podcasts, radio en e-boeken, een
AI-maatje met duidelijke grenzen, denksport en een privé schrijfhoek. Alles in één
toetsenbord-gestuurde terminal-app, zonder bureaublad en zonder browser.

**Zonder spelletjes, feeds, aanbevelingen, autoplay, likes of streaks.** Het kind kiest zelf,
uit een collectie die de ouders samenstellen. Zelfs het AI-maatje praat alleen als het iets
gevraagd wordt, houdt gesprekken kort en helpt nadenken in plaats van voor je te denken.

Het project begon op een oude Asus EeePC, vandaar de "PC" in de naam.

*English version: [README.md](README.md)* · Klikbare mockup: open [`docs/mockup.html`](docs/mockup.html) in een browser.

---

## Uitgangspunten

1. **Geen feeds, geen algoritmes.** Geen "aanbevolen voor jou", geen eindeloos scrollen, geen automatisch doorspelen.
2. **Geen spelletjes, punten of streaks.** Niets wat beloont voor "nog even".
3. **Rust in de avond.** Het avondslot hoort bij het systeem: na bedtijd blijven alleen lezen en een luisterboek met slaaptimer over (in te stellen).
4. **Eerlijk.** Het kind weet wat ouders kunnen zien. De schrijfhoek is privé.
5. **Zelfstandig.** Werkt los van andere systemen, met speakers eraan. Koppelen aan een NAS of Music Assistant kan, maar hoeft niet.
6. **Licht.** Moet vlot draaien op een 32-bit netbook met 1 GB geheugen.

### Ook Spotify zonder algoritme

- **Geen autoplay.** Unscroll bepaalt zelf de wachtrij: een album speelt af en stopt dan. Bij librespot staat autoplay uit.
- **Geen aanbevelingen.** De app vraagt nooit om "Made for you", Daily Mix, Discover Weekly of radio op basis van een nummer.
- **Alleen goedgekeurde albums en lijsten.** Zoeken in heel Spotify staat standaard uit, omdat ook de volgorde van zoekresultaten door een algoritme wordt bepaald.
- **Tip voor ouders:** gebruik een eigen Spotify-profiel voor het kind en keur geen door Spotify gemaakte mixen goed. Spotify leert nog steeds van wat er geluisterd wordt; dat zie je niet op Unscroll, wel in de officiële app.

## Wat erin zit

| Onderdeel | Inhoud |
|---|---|
| Start | Begroeting, menu, avondslot, eigen naam in blokletters |
| Talen | Engels (standaard) en Nederlands, te kiezen bij Systeem. Nieuwe talen in `app/i18n.go` |
| Systeem | Naam computer, eigen naam, naam en gezicht van het maatje, 5 kleurthema's, wifi, accu, volume met maximum, hulp vragen (ntfy) |
| Avondslot | Aparte tijden voor school- en weekendavonden, openingstijd, wat er na het slot nog mag, pincode voor ouders, pauzeren voor 30 min, 2 uur, tot morgenochtend of tot je hem weer aanzet |
| Updates | Nieuwe versies via GitHub-releases, met checksum-controle en terugval; de app herstart zelf op een rustig moment (avondslot of midden in de nacht, als er niets speelt), of meteen met `R` in de ouderstand |
| Luisteren | Navidrome, radio met draaiknop, luisterboeken en podcasts uit Audiobookshelf met voortgang, slaaptimer |
| Lezen | E-boeken (EPUB) uit Audiobookshelf, rustige leesweergave, ook offline, voortgang gaat mee |
| Vandaag | Weer met regen per uur en praktische tips (Open-Meteo, geen sleutel), zon op en onder, maanfase, seizoen, de volgende jaarfeesten |
| Spotify | Alleen goedgekeurde afspeellijsten, expliciete nummers gefilterd, autoplay uit (librespot) |
| Vraag het maatje | AI via de Anthropic-API met dagbudget, filters en weekoverzicht voor ouders |
| Ontdekken | Dier van de dag (vaste lijst, tekst van Wikipedia), computerlab (de terminal leren met missies op een oefencomputer), privé natuurdagboek, sterrenhemel (planeten, maan en sterrenbeelden van vanavond, een hemelkaart, wanneer het ISS bij jou overkomt, en 21 korte uitleggen over sterrenkunde) |
| Programmeren | [Hedy](https://hedy.org): de echte Hedy-taal, niveau voor niveau richting Python, met Hedy's eigen opdrachten in het Nederlands en Engels, en een schildpad die met tekens tekent |
| Denksport | Tafels van 1 tot en met 12 oefenen zonder klok (overzicht van wat erin zit, lastige sommen komen terug), rekenmachine met knoppen (basis en wetenschappelijk: sin, cos, tan, √, log, ln, π, geheugen) en een rekenmachine om sommen te typen, met exacte breuken en procenten, sudoku (elke keer een nieuwe, drie niveaus), raadsels en getallenrijen, schaken: 2000 puzzels uit de [puzzeldatabase van Lichess](https://database.lichess.org/) (CC0), tegen [Stockfish](https://stockfishchess.org/) op acht niveaus, of met z'n tweeën |
| Typcursus | Rustige lessen zonder klok of score (de Nederlandse juniorlessen van GNU Typist, geïnstalleerd uit Debian), een toetsenbord in beeld met de volgende toets en vinger, daarna eigen zinnen |
| Schrijfhoek | Eenvoudige takenlijst, woordenboek met hulp bij spelling (Wiktionary), je eigen verhalen teruglezen; privé |
| Ouderinstellingen | In de ouderstand (Systeem, `o` en de pincode, dan `i`): avondslot, volume, grenzen en model van het maatje, de woonplaats voor het weer, Spotify; wijzigingen werken meteen |

## Nog op de lijst

Ideeën en plannen die nog niet gebouwd zijn, ongeveer in volgorde:

- **Berichten naar ouders:** Signal via [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) op een thuisserver (eigen nummer voor het kind): korte berichtjes alleen naar contacten in ouders.yaml, antwoorden als rustige lijst
- **Diagnose in de ouderstand:** per dienst of hij werkt, met de precieze foutmelding.
- **Veilig bewaren:** bestanden zo wegschrijven dat een plotselinge stroomuitval niets kapotmaakt.
- **Ochtend- en avondlijstje** dat elke dag opnieuw begint.
- **Verlanglijst voor de muziekbibliotheek:** het kind vraagt een album aan, ouders krijgen een melding.
- **Weekoverzicht voor ouders** via ntfy.
- **Hedy-galerij:** je eigen programma's terugzien en opnieuw draaien.
- **Helpscherm per tabblad** en een instelling voor de lettergrootte.

## Geschikte apparaten

Alles wat Debian 12 draait. Getest en bedoeld voor:

| Apparaat | Architectuur | Opmerking |
|---|---|
| Asus EeePC en andere netbooks | 386 (32-bit) | Het startpunt van dit project. 1 GB geheugen is genoeg. |
| Oude laptops (ThinkPad, Dell, HP) | amd64 | Vaak voor een paar tientjes tweedehands te vinden |
| Ex-school Chromebooks met Intel | amd64 | Eerst ombouwen met de firmware van MrChromebox |
| Raspberry Pi 4 / 5 met scherm en toetsenbord | arm64 | Raspberry Pi OS Lite (Bookworm) |
| Raspberry Pi 400 / 500 | arm64 | Computer in een toetsenbord: alleen nog een scherm erbij |
| Draagbare Pi-terminals (uConsole, DevTerm) | arm64 | Een echte "cyberdeck" |

## Installeren

1. Installeer **Debian 12** minimaal (alleen "SSH-server" en "standaard systeemhulpmiddelen", geen bureaublad).
2. Zet deze repository op het apparaat en draai als root:
   ```bash
   sudo bash install.sh
   ```
3. Stel pincode, avondslot en geheimen lokaal in met de ouderhulp, en herstart:
   ```bash
   sudo unscroll-ouder
   ```

Na het opstarten logt het kinderaccount automatisch in en start de app op het hele scherm
(`cage` met de terminal `foot`). Er is geen bureaublad, geen browser en geen vrije shell.

## Updates

- **Een nieuwe versie uitbrengen:** verhoog het nummer in `VERSION` en push naar `main`. GitHub Actions test alles, bouwt de app voor 386, amd64, arm64 en arm (met een `sha256`-bestand per binary) en maakt de release aan.
- Een nummer met een streepje (`0.2.0-test`) wordt een pre-release. Die gaat alleen naar apparaten met `KANAAL="test"` in `/etc/unscroll/update.conf`.
- Het apparaat kijkt elke 15 minuten of er een nieuwe release is (`unscroll-update.timer`), controleert de checksum en of de nieuwe versie start, en zet hem klaar. De vorige versie blijft bewaard. De app herstart daarna zelf op een rustig moment: tijdens het avondslot of midden in de nacht, als er een kwartier niemand aan het toetsenbord zat, er niets speelt en er geen onbewaard werk open kan staan. In de ouderstand laat het tabblad Systeem zien dat er een nieuwe versie klaarstaat, en met `R` herstart je meteen.
- De scripts (sessie, updater, ouderhulp) komen mee in een `unscroll-scripts.tar.gz` met checksum en worden samen met de app geïnstalleerd (vanaf v0.2.0).
- Is de repository privé, zet dan een fine-grained token met alleen-lezen rechten in `/etc/unscroll/geheimen.env` (de ouderhulp vraagt erom).

## Luisteren

Druk op `1` voor Luisteren en op `Tab` om te wisselen tussen Muziek, Luisterboeken, Podcasts en Radio.
Alles speelt via `mpv`, zonder beeld. Er is geen autoplay: een album speelt tot het einde en dan is het stil.

| Bron | Nodig | Instellen met |
|---|---|
| Muziek | [Navidrome](https://www.navidrome.org/) met een eigen gebruiker voor het kind | `sudo unscroll-ouder` (adres, gebruiker, wachtwoord) |
| Luisterboeken en podcasts | [Audiobookshelf](https://www.audiobookshelf.org/) met een account voor het kind, zodat de voortgang van het kind is | `sudo unscroll-ouder` (adres en API-token of API-sleutel) |
| Radio | niets: NPO Radio 1, 2, 3FM, Klassiek en BBC World Service zitten erin | een eigen lijst onder `radio:` in `/etc/unscroll/ouders.yaml` |

Toetsen: `↑ ↓` kiezen, `Enter` openen of afspelen, `Backspace` terug, `/` zoeken, `spatie` pauze, `x` stop,
`n` `p` volgend of vorig nummer, `← →` 30 seconden terug of vooruit (bij de radio: aan de knop draaien),
`s` slaaptimer (15, 30, 45, 60 minuten), `+ -` volume tot het maximum van de ouders.

Tijdens het avondslot blijven alleen luisterboeken over (als `luisterboek_mag` aan staat), met een
slaaptimer van 30 minuten die vanzelf start. Muziek, radio en podcasts stoppen als het avondslot begint.

Ouders kunnen het avondslot pauzeren in de ouderstand (`8`, dan `o` en de pincode): `t` 30 minuten,
`u` 2 uur, `m` tot morgenochtend, `x` tot je hem weer aanzet, `a` weer aan. De ouderstand gaat na
10 minuten zonder toetsaanslag vanzelf uit.

## Instellingen

| Bestand | Van wie | Inhoud |
|---|---|
| `~/.config/unscroll/instellingen.json` | het kind | naam, maatje, gezicht, kleurthema |
| `/etc/unscroll/ouders.yaml` | ouders (root) | avondslot, pincode, maximaal volume, ntfy-topic |
| `/etc/unscroll/update.conf` | ouders (root) | repository en updatekanaal |
| `/etc/unscroll/geheimen.env` | ouders (root) | tokens en API-sleutels, nooit in git |

## Spotify zonder algoritme

Vraagt Spotify Premium (Duo werkt; met Family krijgt het kind een eigen account en loopt zijn
luistergedrag niet door het jouwe). Het geluid komt uit [librespot](https://github.com/librespot-org/librespot),
een Spotify Connect-speaker die de updater installeert (`/usr/local/bin/librespot`, niet voor 32-bit ARM).
De app bestuurt hem via de Spotify Web API en speelt alleen wat jullie hebben goedgekeurd.

Eenmalig, op een telefoon of computer:
1. Maak een app aan op [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard), vink Web API aan
   en voeg de redirect URI `http://127.0.0.1:8898/callback` toe. Kopieer de Client ID.
2. Doe op het apparaat `sudo unscroll-ouder`, vul de Client ID in, log in (de ouderhulp toont een link;
   je komt uit op een pagina die niet laadt, kopieer dat adres terug) en koppel de speaker op dezelfde manier.
   Gebruik voor allebei hetzelfde Spotify-account.
3. Keur albums en playlists goed: zet hun links onder `spotify: goedgekeurd:` in `/etc/unscroll/ouders.yaml`.

Met `zoeken: true` (vraagt `sudo unscroll-ouder`) kan het kind ook zelf artiesten en albums zoeken,
alle albums van een artiest bekijken en favorieten bewaren (`f`), zonder aanbevelingen, radio of autoplay.
Expliciete nummers en artiesten onder `geblokkeerd:` blijven weg; ouders zien in de ouderstand wat er de
afgelopen week is gezocht en geluisterd (`w` in de Spotify-lijst).

In de app: `1` Luisteren, `Tab` naar Spotify. Een goedgekeurd album of playlist speelt op volgorde en stopt dan.
Begint er iets anders te spelen op de speaker (autoplay, of iemand die hem vanaf een ander apparaat overneemt),
dan pauzeert de app het.

## Het AI-maatje

Druk op `3` (het tabblad heeft de naam van het maatje). Zet de API-sleutel en kies het model met `sudo unscroll-ouder`;
maak hem aan op [console.anthropic.com](https://console.anthropic.com) en stel daar een maandlimiet in.
Het maatje antwoordt standaard met `claude-opus-5-5` en elke vraag en elk antwoord wordt gecontroleerd
door `claude-haiku-5-5`. Beide kun je wijzigen onder `maatje:` in `/etc/unscroll/ouders.yaml`
(bijvoorbeeld `model: "claude-sonnet-5-5"` om minder uit te geven).

Het maatje is een hulpje, geen vriend: het praat alleen als het iets gevraagd wordt en helpt
nadenken in plaats van voor het kind te denken. De grenzen gaan per soort gebruik:

| Modus | Hoe | Standaardgrens |
|---|---|
| Huiswerkcoach en overhoren | Sessie rond één opdracht: eerst zelf proberen, hints in stappen, voorbeeld met andere getallen, afsluiten met "wat heb je geleerd?" | 2 sessies per dag, 40 beurten per sessie |
| Samen een verhaal | Om de beurt schrijven, wordt bewaard in de schrijfhoek | 1 sessie per dag |
| Vrije vragen | Nieuwsgierigheid: "hoe werkt het?" | 15 per dag |
| Alles samen | | 45 minuten per dag, pauze na 20 minuten, dicht tijdens het avondslot |

Veiligheid: persoonsgegevens (e-mail, telefoon, adres, postcode, IBAN, de eigen naam van het kind)
worden op het apparaat weggefilterd voordat er iets wordt verstuurd, elke vraag en elk antwoord krijgt
een tweede controle op geschiktheid, zware onderwerpen worden doorverwezen naar de ouders (met een
ntfy-melding zonder de vraag zelf), en `Tab` op een lege regel vraagt "klopt dit wel?", wat leert dat
AI fouten kan maken.

Ouders zien de vragen van de afgelopen week in de ouderstand: open het tabblad van het maatje en druk
op `w`. Samen geschreven verhalen komen in de schrijfhoek (`~/.local/share/unscroll/verhalen`); ouders
zien alleen dat er een verhaal is geschreven, niet het verhaal zelf.

Ouders stellen de grenzen in in `/etc/unscroll/ouders.yaml` (onderdeel `maatje`; 0 betekent geen grens). Verhalen en huiswerk kun je later hervatten. Ouders lezen huiswerk terug in de ouderstand (`h` in het tabblad van het maatje) of via ssh met `sudo unscroll-ouder huiswerk`. De standaardwaarden en de instructies
voor het maatje zitten in de app en worden via updates bijgesteld op basis van ervaringen.

## Programmeren met Hedy

Ontdekken (`4`) → Programmeren gebruikt [Hedy](https://hedy.org) (Universiteit Leiden, EUPL-1.2): het taalpakket
[hedylang](https://pypi.org/project/hedylang/) en de opdrachten ("avonturen") uit
[hedyorg/hedy](https://github.com/hedyorg/hedy). Releases leveren ze mee als `unscroll-hedy.tar.gz` (gebouwd door
`hedy/bouw-pakket.sh` in Debian 12, met twee kleine aanpassingen zodat het op Python 3.11 draait); de updater
installeert het in `/opt/unscroll/hedy`. Programma's draaien als apart Python-proces met een tijd- en
geheugengrens, zonder toegang tot bestanden en zonder andere imports dan `random`, `time` en `math`. Tekenen met
de schildpad geeft een plaatje van braille-tekens; toetsen indrukken en muziek kunnen nog niet. De eerste keer
op een niveau kan op een oude netbook even duren, omdat Hedy dat niveau één keer voorbereidt.

## Wachtwoorden en sleutels

**Geheimen komen nooit in deze repository.** Pincode, wachtwoorden, gebruikersnamen, tokens en
API-sleutels stel je lokaal in op het apparaat met de ouderhulp:

```bash
sudo unscroll-ouder
```

Die schrijft naar `/etc/unscroll/ouders.yaml` en `/etc/unscroll/geheimen.env` (alleen leesbaar
voor root en de app). In git staan alleen voorbeeldbestanden met lege waarden. GitHub Actions
controleert bij elke push dat er geen echte instellingen of sleutels in de repository staan.

## Ontwikkelen

```bash
cd app
go test ./...
go run .                      # draait in je eigen terminal
UNSCROLL_OUDERS=../config/ouders.example.yaml go run .
CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -o unscroll .
```

De app is geschreven in Go met [Bubble Tea](https://github.com/charmbracelet/bubbletea) en
[Lip Gloss](https://github.com/charmbracelet/lipgloss), zodat het één klein bestand is dat ook
op oude 32-bit hardware vlot draait.

## Mappen

```
app/        de terminal-app (Go)
scripts/    unscroll-sessie (kioskstart), unscroll-update (updater), unscroll-ouder (ouderhulp)
systemd/    timer en service voor de updater
config/     voorbeelden van de instellingen en de terminal-configuratie
docs/       klikbare mockup
install.sh  installatie op een verse Debian 12
```

## Licentie

Unscroll PC valt onder de [MIT-licentie](LICENSE). Onderdelen die het gebruikt houden hun eigen licentie:
[Hedy](https://github.com/hedyorg/hedy) (EUPL-1.2, meegeleverd in het Hedy-pakket met het licentiebestand),
de schaakpuzzels uit de [puzzeldatabase van Lichess](https://database.lichess.org/) (CC0),
de satellietbibliotheek [go-satellite](https://github.com/joshuaferrara/go-satellite) (BSD-2) met baangegevens van het ISS van [CelesTrak](https://celestrak.org/),
en Debian-pakketten die apart worden geïnstalleerd en geen deel zijn van deze repository, zoals
[Stockfish](https://stockfishchess.org/) (GPL-3) en de typlessen van [GNU Typist](https://www.gnu.org/software/gtypist/) (GPL).
