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

## Wat erin zit (en wat er nog komt)

| Onderdeel | Inhoud | Status |
|---|---|---|
| Start | Begroeting, menu, avondslot, eigen naam in blokletters | ✅ fase 0 |
| Talen | Engels (standaard) en Nederlands, te kiezen bij Systeem. Nieuwe talen in `app/i18n.go` | ✅ fase 0 |
| Systeem | Naam computer, eigen naam, naam en gezicht van het maatje, 5 kleurthema's, wifi, accu, volume met maximum, hulp vragen (ntfy) | ✅ fase 0 |
| Avondslot | Aparte tijden voor school- en weekendavonden, openingstijd, wat er na het slot nog mag, pincode voor ouders, pauzeren voor 30 min, 2 uur, tot morgenochtend of tot je hem weer aanzet | ✅ fase 0 |
| Updates | Nieuwe versies via GitHub-releases, met checksum-controle en terugval | ✅ fase 0 |
| Luisteren | Navidrome, radio met draaiknop, luisterboeken en podcasts uit Audiobookshelf met voortgang, slaaptimer | ✅ fase 1 |
| Lezen | E-boeken (EPUB) uit Audiobookshelf, rustige leesweergave, ook offline, voortgang gaat mee | ✅ fase 2 |
| Vandaag | Weer, wekker, maan, seizoenen en jaarfeesten | ⏳ fase 2 |
| Spotify | Alleen goedgekeurde afspeellijsten, expliciete nummers gefilterd, autoplay uit (librespot) | ✅ fase 3 |
| Vraag het maatje | AI via de Anthropic-API met dagbudget, filters en weekoverzicht voor ouders | ✅ fase 4 |
| Ontdekken | Dier van de dag (vaste lijst, tekst van Wikipedia), computerlab (de terminal leren met missies op een oefencomputer), privé natuurdagboek | ✅ fase 5 |
| Python | Programmeren met alleen je eigen kamerlamp als echte uitvoer | ⏳ later |
| Denksport | Sudoku, schaakpuzzels, raadsels. Geen timer, geen score | ⏳ fase 5 |
| Typcursus | Rustige typlessen: eerst de thuisrij, korte lessen per dag, oefenen met eigen zinnen en verhalen, vooruitgang zonder scores of reeksen | ⏳ later |
| Schrijfhoek | Privé dagboek en verhalen | ⏳ fase 5 |

## Geschikte apparaten

Alles wat Debian 12 draait. Getest en bedoeld voor:

| Apparaat | Architectuur | Opmerking |
|---|---|---|
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
- Het apparaat kijkt elke 15 minuten of er een nieuwe release is (`unscroll-update.timer`), controleert de checksum en of de nieuwe versie start, en zet hem klaar. De vorige versie blijft bewaard. De nieuwe versie is actief na een herstart van de app (of van het apparaat).
- De scripts (sessie, updater, ouderhulp) komen mee in een `unscroll-scripts.tar.gz` met checksum en worden samen met de app geïnstalleerd (vanaf v0.2.0).
- Is de repository privé, zet dan een fine-grained token met alleen-lezen rechten in `/etc/unscroll/geheimen.env` (de ouderhulp vraagt erom).

## Luisteren (fase 1)

Druk op `1` voor Luisteren en op `Tab` om te wisselen tussen Muziek, Luisterboeken, Podcasts en Radio.
Alles speelt via `mpv`, zonder beeld. Er is geen autoplay: een album speelt tot het einde en dan is het stil.

| Bron | Nodig | Instellen met |
|---|---|---|
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
|---|---|---|
| `~/.config/unscroll/instellingen.json` | het kind | naam, maatje, gezicht, kleurthema |
| `/etc/unscroll/ouders.yaml` | ouders (root) | avondslot, pincode, maximaal volume, ntfy-topic |
| `/etc/unscroll/update.conf` | ouders (root) | repository en updatekanaal |
| `/etc/unscroll/geheimen.env` | ouders (root) | tokens en API-sleutels, nooit in git |

## Spotify zonder algoritme (fase 3)

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

## Het AI-maatje (fase 4)

Druk op `3` (het tabblad heeft de naam van het maatje). Zet de API-sleutel en kies het model met `sudo unscroll-ouder`;
maak hem aan op [console.anthropic.com](https://console.anthropic.com) en stel daar een maandlimiet in.
Het maatje antwoordt standaard met `claude-opus-5-5` en elke vraag en elk antwoord wordt gecontroleerd
door `claude-haiku-5-5`. Beide kun je wijzigen onder `maatje:` in `/etc/unscroll/ouders.yaml`
(bijvoorbeeld `model: "claude-sonnet-5-5"` om minder uit te geven).

Het maatje is een hulpje, geen vriend: het praat alleen als het iets gevraagd wordt en helpt
nadenken in plaats van voor het kind te denken. De grenzen gaan per soort gebruik:

| Modus | Hoe | Standaardgrens |
|---|---|---|
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

Nog te kiezen.
