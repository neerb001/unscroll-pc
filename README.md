# Unscroll PC

**A calm computer for curious kids. No feeds, no algorithms.**
*Nothing pulls at you. You pull.*

Unscroll verandert een oude laptop, netbook of Raspberry Pi in een rustige luister- en
ontdekcomputer voor kinderen: muziek, luisterboeken, podcasts, radio en e-boeken, een
AI-maatje met duidelijke grenzen, denksport en een privé schrijfhoek. Alles in één
toetsenbord-gestuurde terminal-app, zonder bureaublad en zonder browser.

**Zonder spelletjes, feeds, aanbevelingen, autoplay, likes of streaks.** Het kind kiest zelf,
uit een collectie die de ouders samenstellen. Zelfs het AI-maatje praat alleen als het iets
gevraagd wordt, houdt gesprekken kort en helpt nadenken in plaats van voor je te denken.

Het project begon op een oude Asus EeePC, vandaar de "PC" in de naam.

> *Unscroll turns an old laptop, netbook or Raspberry Pi into a calm listening and discovery
> computer for kids. Dutch first, translations welcome.*

Klikbare mockup: open [`docs/mockup.html`](docs/mockup.html) in een browser.

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
| Avondslot | Aparte tijden voor school- en weekendavonden, openingstijd, wat er na het slot nog mag, pincode voor ouders | ✅ fase 0 |
| Updates | Nieuwe versies via GitHub-releases, met checksum-controle en terugval | ✅ fase 0 |
| Luisteren | Navidrome, radio met draaiknop, luisterboeken en podcasts uit Audiobookshelf, slaaptimer | ⏳ fase 1 |
| Lezen + Vandaag | E-boeken (EPUB) uit Audiobookshelf, weer, wekker, maan, seizoen en jaarfeesten | ⏳ fase 2 |
| Spotify | Alleen goedgekeurde afspeellijsten, expliciete nummers gefilterd, autoplay uit (librespot) | ⏳ fase 3 |
| Vraag het maatje | AI via de Anthropic-API met dagbudget, filters en weekoverzicht voor ouders | ⏳ fase 4 |
| Ontdekken | Computerlab (Linux leren met missies), Python, dier van de dag, natuurdagboek | ⏳ fase 5 |
| Denksport | Sudoku, schaakpuzzels, raadsels. Geen timer, geen score | ⏳ fase 5 |
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

- Een push van een tag (`v0.2.0`) laat GitHub Actions de app bouwen voor 386, amd64, arm64 en arm, met een `sha256`-bestand per binary.
- Het apparaat kijkt elke 15 minuten of er een nieuwe release is (`unscroll-update.timer`), controleert de checksum en of de nieuwe versie start, en zet hem klaar. De vorige versie blijft bewaard.
- Pre-releases (`v0.2.0-test`) gaan alleen naar apparaten met `KANAAL="test"` in `/etc/unscroll/update.conf`.
- Is de repository privé, zet dan een fine-grained token met alleen-lezen rechten in `/etc/unscroll/geheimen.env`.

## Instellingen

| Bestand | Van wie | Inhoud |
|---|---|---|
| `~/.config/unscroll/instellingen.json` | het kind | naam, maatje, gezicht, kleurthema |
| `/etc/unscroll/ouders.yaml` | ouders (root) | avondslot, pincode, maximaal volume, ntfy-topic |
| `/etc/unscroll/update.conf` | ouders (root) | repository en updatekanaal |
| `/etc/unscroll/geheimen.env` | ouders (root) | tokens en API-sleutels, nooit in git |

## Het AI-maatje (fase 4)

Het maatje is een hulpje, geen vriend: het praat alleen als het iets gevraagd wordt en helpt
nadenken in plaats van voor het kind te denken. De grenzen gaan per soort gebruik:

| Modus | Hoe | Standaardgrens |
|---|---|---|
| Huiswerkcoach en overhoren | Sessie rond één opdracht: eerst zelf proberen, hints in stappen, voorbeeld met andere getallen, afsluiten met "wat heb je geleerd?" | 2 sessies per dag, 40 beurten per sessie |
| Samen een verhaal | Om de beurt schrijven, wordt bewaard in de schrijfhoek | 1 sessie per dag |
| Vrije vragen | Nieuwsgierigheid: "hoe werkt het?" | 15 per dag |
| Alles samen | | 45 minuten per dag, pauze na 20 minuten, dicht tijdens het avondslot |

Veiligheid: persoonsgegevens worden weggefilterd voordat er iets wordt verstuurd, elke vraag
en elk antwoord krijgt een tweede controle op geschiktheid, zware onderwerpen worden
doorverwezen naar de ouders, en een "klopt dit?"-knop leert dat AI fouten kan maken.

Ouders stellen de grenzen in met `sudo unscroll-ouder`. De standaardwaarden en de instructies
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
