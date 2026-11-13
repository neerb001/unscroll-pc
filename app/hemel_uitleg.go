package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Uitleg over sterrenkunde: korte, kloppende stukjes voor een kind van een jaar of elf.
// Elk onderwerp heeft een Nederlandse en een Engelse tekst, plus (bij planeten) een rijtje feiten.

type hemelOnderwerp struct {
	sleutel string
	titel   [2]string // nl, en
	feiten  [2]string
	tekst   [2]string
}

var hemelOnderwerpen = []hemelOnderwerp{
	{"sun", [2]string{"De zon", "The sun"},
		[2]string{"150 miljoen km van ons · 109 keer zo breed als de aarde · 5500 °C aan de buitenkant", "150 million km from us · 109 times as wide as Earth · 5500 °C on the outside"},
		[2]string{
			"De zon is een ster, net als de sterren die je 's nachts ziet. Hij lijkt alleen veel groter en feller omdat hij zo dichtbij staat. Diep in de zon worden kleine deeltjes (waterstof) samengeperst tot helium. Daarbij komt enorm veel energie vrij: dat is het licht en de warmte die wij voelen. Licht van de zon doet ruim 8 minuten over de reis naar de aarde. Kijk nooit recht in de zon, ook niet met een verrekijker: dat maakt je ogen kapot.",
			"The sun is a star, just like the stars you see at night. It only looks much bigger and brighter because it is so close. Deep inside the sun, tiny particles (hydrogen) are squeezed together into helium. That releases a huge amount of energy: the light and warmth we feel. Sunlight takes a little over 8 minutes to reach Earth. Never look straight at the sun, not even with binoculars: it damages your eyes.",
		}},
	{"earth", [2]string{"De aarde", "Earth"},
		[2]string{"draait in 24 uur om zichzelf · in 365¼ dag om de zon · 1 maan", "spins once in 24 hours · goes round the sun in 365¼ days · 1 moon"},
		[2]string{
			"De aarde is de derde planeet vanaf de zon en de enige waarvan we zeker weten dat er leven is. Ze draait als een tol om haar eigen as: de kant die naar de zon kijkt heeft dag, de andere kant nacht. Daarom lijken de zon en de sterren langs de hemel te schuiven, van oost naar west, terwijl eigenlijk wij draaien. Elk jaar maakt de aarde één rondje om de zon. Omdat dat net iets langer dan 365 dagen duurt, hebben we om de vier jaar een schrikkeldag.",
			"Earth is the third planet from the sun and the only one where we know for sure there is life. It spins like a top on its own axis: the side facing the sun has day, the other side night. That is why the sun and stars seem to slide across the sky from east to west, while really we are the ones turning. Every year Earth makes one trip around the sun. Because that takes a little longer than 365 days, we get a leap day every four years.",
		}},
	{"moon", [2]string{"De maan en haar fasen", "The moon and its phases"},
		[2]string{"384.000 km van ons · draait in 27⅓ dag om de aarde · van nieuwe maan tot nieuwe maan: 29½ dag", "384,000 km from us · goes round Earth in 27⅓ days · new moon to new moon: 29½ days"},
		[2]string{
			"De maan geeft zelf geen licht: ze is een grote grijze rotsbol die door de zon wordt beschenen. De helft die naar de zon kijkt is altijd licht. Terwijl de maan om de aarde draait, zien wij steeds een ander stuk van die lichte helft. Zo ontstaan de fasen: nieuwe maan (niets te zien), wassende maan (de rechterkant wordt licht), volle maan en afnemende maan. In Nederland geldt: is de rechterkant licht, dan groeit hij; kun je er met een streepje een a van maken (bolletje links), dan neemt hij af. De maan laat ons altijd dezelfde kant zien.",
			"The moon makes no light of its own: it is a big grey ball of rock lit up by the sun. The half facing the sun is always bright. As the moon travels around Earth, we see a different part of that bright half. That gives the phases: new moon (nothing to see), waxing moon (growing), full moon and waning moon (shrinking). In the northern hemisphere a waxing moon is lit on the right side, a waning moon on the left. The moon always shows us the same face.",
		}},
	{"mercury", [2]string{"Mercurius", "Mercury"},
		[2]string{"58 miljoen km van de zon · een jaar duurt 88 dagen · geen manen", "58 million km from the sun · a year lasts 88 days · no moons"},
		[2]string{
			"Mercurius is de kleinste planeet en staat het dichtst bij de zon. Er is bijna geen lucht, dus overdag wordt het er 430 °C en 's nachts zakt de temperatuur naar -180 °C. Omdat hij zo dicht bij de zon staat, zie je Mercurius alleen kort na zonsondergang of vlak voor zonsopkomst, laag bij de horizon. Hij is vernoemd naar de snelle boodschapper van de Romeinse goden: geen planeet gaat sneller rond de zon.",
			"Mercury is the smallest planet and the closest to the sun. It has almost no air, so in the daytime it reaches 430 °C and at night it drops to -180 °C. Because it is so close to the sun, you only see Mercury shortly after sunset or just before sunrise, low over the horizon. It is named after the fast messenger of the Roman gods: no planet goes round the sun faster.",
		}},
	{"venus", [2]string{"Venus", "Venus"},
		[2]string{"108 miljoen km van de zon · een jaar duurt 225 dagen · geen manen", "108 million km from the sun · a year lasts 225 days · no moons"},
		[2]string{
			"Venus is bijna even groot als de aarde, maar je zou er niet willen wonen. Een dikke laag wolken houdt de warmte vast, waardoor het er 465 °C is: heter dan op Mercurius. Die wolken kaatsen het zonlicht juist heel goed terug. Daarom is Venus na de maan het felste lichtpunt aan de nachthemel. Mensen noemen haar de avondster of de morgenster, al is het helemaal geen ster. Een dag op Venus duurt langer dan een jaar daar, en ze draait de andere kant op dan de meeste planeten.",
			"Venus is almost the same size as Earth, but you would not want to live there. A thick layer of clouds traps the heat, so it is 465 °C: hotter than Mercury. Those clouds reflect sunlight very well, which makes Venus the brightest point in the night sky after the moon. People call it the evening star or morning star, even though it is not a star at all. A day on Venus lasts longer than its year, and it spins the opposite way to most planets.",
		}},
	{"mars", [2]string{"Mars", "Mars"},
		[2]string{"228 miljoen km van de zon · een dag duurt 24 uur en 37 minuten · een jaar 687 dagen · 2 manen", "228 million km from the sun · a day lasts 24 hours 37 minutes · a year 687 days · 2 moons"},
		[2]string{
			"Mars heet de rode planeet omdat de grond vol roest zit. Je ziet hem aan de hemel ook echt als een oranjerood puntje. Er staat de hoogste berg van het zonnestelsel, Olympus Mons: bijna drie keer zo hoog als de Mount Everest. Mars is koud en de lucht is erg dun. Er rijden robotwagentjes rond, zoals Curiosity en Perseverance, die zoeken naar sporen van water en misschien heel oud leven. De twee kleine maantjes heten Phobos en Deimos.",
			"Mars is called the red planet because its ground is full of rust. In the sky you really see it as an orange-red dot. It has the highest mountain in the solar system, Olympus Mons: almost three times as high as Mount Everest. Mars is cold and its air is very thin. Robot rovers such as Curiosity and Perseverance drive around there, looking for signs of water and maybe very old life. Its two small moons are called Phobos and Deimos.",
		}},
	{"jupiter", [2]string{"Jupiter", "Jupiter"},
		[2]string{"778 miljoen km van de zon · een jaar duurt bijna 12 aardse jaren · meer dan 90 manen", "778 million km from the sun · a year lasts almost 12 Earth years · more than 90 moons"},
		[2]string{
			"Jupiter is de grootste planeet: alle andere planeten passen er samen in. Hij heeft geen vaste grond, hij bestaat vooral uit gas. Op Jupiter woedt al honderden jaren een storm die groter is dan de aarde: de Grote Rode Vlek. Met een gewone verrekijker zie je vaak vier puntjes naast Jupiter. Dat zijn zijn grootste manen (Io, Europa, Ganymedes en Callisto), die Galileo Galilei in 1610 ontdekte. Elke avond staan ze op een andere plek.",
			"Jupiter is the biggest planet: all the other planets together would fit inside it. It has no solid ground; it is mostly gas. A storm bigger than Earth has been raging on Jupiter for hundreds of years: the Great Red Spot. With ordinary binoculars you can often see four dots next to Jupiter. Those are its largest moons (Io, Europa, Ganymede and Callisto), discovered by Galileo Galilei in 1610. Every night they are in a different place.",
		}},
	{"saturn", [2]string{"Saturnus", "Saturn"},
		[2]string{"1,4 miljard km van de zon · een jaar duurt 29 aardse jaren · meer dan 250 manen", "1.4 billion km from the sun · a year lasts 29 Earth years · more than 250 moons"},
		[2]string{
			"Saturnus is beroemd om zijn ringen. Die bestaan uit miljarden stukken ijs en steen, van zandkorrels tot huizen groot. De ringen zijn heel breed maar meestal maar een paar tientallen meters dik. Saturnus is een gasreus, net als Jupiter, en zo licht voor zijn grootte dat hij in een (heel, heel grote) badkuip zou blijven drijven. Met het blote oog is hij een rustig geel lichtpuntje; met een kleine telescoop zie je de ringen al.",
			"Saturn is famous for its rings. They are made of billions of pieces of ice and rock, from grains of sand to the size of houses. The rings are very wide but usually only a few tens of metres thick. Saturn is a gas giant like Jupiter, and so light for its size that it would float in a (very, very large) bathtub. To the naked eye it is a steady yellow dot; with a small telescope you can already see the rings.",
		}},
	{"uranus", [2]string{"Uranus", "Uranus"},
		[2]string{"2,9 miljard km van de zon · een jaar duurt 84 aardse jaren · minstens 28 manen", "2.9 billion km from the sun · a year lasts 84 Earth years · at least 28 moons"},
		[2]string{
			"Uranus is een blauwgroene ijsreus. Het rare aan Uranus: hij ligt op zijn kant. Waarschijnlijk is hij lang geleden ergens tegenaan gebotst. Daardoor is het aan de ene pool 42 jaar lang dag en daarna 42 jaar lang nacht. Uranus is zo ver weg dat je hem met het blote oog bijna niet kunt zien. Hij werd in 1781 ontdekt door William Herschel, als eerste planeet die met een telescoop werd gevonden.",
			"Uranus is a blue-green ice giant. The odd thing about Uranus: it lies on its side. Probably it crashed into something long ago. Because of that, one pole has 42 years of daylight and then 42 years of night. Uranus is so far away that you can barely see it with the naked eye. William Herschel discovered it in 1781, the first planet ever found with a telescope.",
		}},
	{"neptune", [2]string{"Neptunus", "Neptune"},
		[2]string{"4,5 miljard km van de zon · een jaar duurt 165 aardse jaren · 16 manen", "4.5 billion km from the sun · a year lasts 165 Earth years · 16 moons"},
		[2]string{
			"Neptunus is de verste planeet, diepblauw en ijskoud. Er waaien de hardste winden van het zonnestelsel, meer dan 2000 km per uur. Neptunus werd eerst met rekenwerk gevonden: sterrenkundigen zagen dat Uranus een beetje werd weggetrokken en rekenden uit waar een onbekende planeet moest staan. In 1846 keek iemand op die plek, en daar stond hij. Je hebt een telescoop nodig om hem te zien.",
			"Neptune is the farthest planet, deep blue and freezing cold. It has the strongest winds in the solar system, more than 2000 km per hour. Neptune was found by maths first: astronomers saw that Uranus was being pulled slightly off course and worked out where an unknown planet had to be. In 1846 someone looked at that spot, and there it was. You need a telescope to see it.",
		}},
	{"pluto", [2]string{"Pluto en de dwergplaneten", "Pluto and the dwarf planets"},
		[2]string{"Pluto: 5,9 miljard km van de zon · een jaar duurt 248 aardse jaren · kleiner dan onze maan", "Pluto: 5.9 billion km from the sun · a year lasts 248 Earth years · smaller than our moon"},
		[2]string{
			"Pluto werd in 1930 ontdekt en heette 76 jaar lang de negende planeet. Later vonden sterrenkundigen nog veel meer van zulke kleine ijswereldjes in hetzelfde gebied, zoals Eris, Makemake en Haumea. Daarom bedachten ze in 2006 een nieuwe groep: dwergplaneten. Die zijn rond, draaien om de zon, maar hebben hun baan niet schoongeveegd. In 2015 vloog de ruimtesonde New Horizons langs Pluto en zag er een groot ijsveld in de vorm van een hart.",
			"Pluto was discovered in 1930 and was called the ninth planet for 76 years. Later astronomers found many more small icy worlds in the same area, such as Eris, Makemake and Haumea. So in 2006 they created a new group: dwarf planets. They are round and orbit the sun, but have not swept their orbit clear. In 2015 the space probe New Horizons flew past Pluto and saw a big ice plain shaped like a heart.",
		}},
	{"seasons", [2]string{"De seizoenen", "The seasons"},
		[2]string{"de aardas staat 23,4° scheef", "Earth's axis is tilted 23.4°"},
		[2]string{
			"Veel mensen denken dat het zomer is omdat we dan dichter bij de zon staan, maar dat klopt niet. Het komt doordat de aarde scheef staat. In juni leunt ons deel van de aarde naar de zon toe: de zon klimt hoog, de dagen zijn lang en het zonlicht valt steil naar beneden, zodat het meer verwarmt. In december leunen we van de zon af: de zon blijft laag en de dagen zijn kort. In Australië is het dan juist zomer. Op 21 juni is de langste dag, rond 21 december de kortste.",
			"Many people think it is summer because we are closer to the sun then, but that is not true. It happens because Earth is tilted. In June our part of the world leans towards the sun: the sun climbs high, days are long and sunlight comes down steeply, so it heats more. In December we lean away from the sun: it stays low and days are short. In Australia it is summer then. Around 21 June is the longest day, around 21 December the shortest.",
		}},
	{"stars", [2]string{"Sterren", "Stars"},
		[2]string{"de dichtstbijzijnde ster na de zon: Proxima Centauri, 4,2 lichtjaar", "the nearest star after the sun: Proxima Centauri, 4.2 light years"},
		[2]string{
			"Elke ster is een zon, vaak veel groter of kleiner dan de onze. Ze zijn zo ver weg dat ze alleen als puntjes te zien zijn. Kijk eens goed naar de kleuren: blauwwitte sterren zijn het heetst, rode sterren het koelst. Betelgeuze in Orion is rood, Rigel in dezelfde Orion blauwwit. Sterren twinkelen omdat hun licht door onze bewegende lucht moet. Planeten twinkelen bijna niet: zo kun je ze uit elkaar houden. Sterren worden geboren in grote wolken gas en stof en kunnen miljarden jaren schijnen.",
			"Every star is a sun, often much bigger or smaller than ours. They are so far away that we only see them as points. Look closely at their colours: blue-white stars are the hottest, red stars the coolest. Betelgeuse in Orion is red, Rigel in the same Orion blue-white. Stars twinkle because their light has to pass through our moving air. Planets hardly twinkle: that is how you can tell them apart. Stars are born in huge clouds of gas and dust and can shine for billions of years.",
		}},
	{"polaris", [2]string{"De Poolster vinden", "Finding the Pole Star"},
		[2]string{"88 officiële sterrenbeelden · de Poolster staat bijna precies boven de noordpool", "88 official constellations · the Pole Star sits almost exactly above the North Pole"},
		[2]string{
			"Sterrenbeelden zijn plaatjes die mensen al duizenden jaren tussen de sterren tekenen. De sterren in één beeld staan vaak helemaal niet dicht bij elkaar; ze lijken alleen van hieruit in dezelfde richting te staan. Zo vind je de Poolster: zoek de Grote Beer (een steelpan van zeven sterren). Trek een lijn door de twee sterren aan de voorkant van de pan en ga vijf keer zo ver omhoog. Daar staat de Poolster. Kijk je ernaar, dan kijk je naar het noorden. Alle andere sterren draaien er 's nachts langzaam omheen.",
			"Constellations are pictures people have drawn between the stars for thousands of years. The stars in one constellation are often not close together at all; they only seem to point in the same direction from here. To find the Pole Star: look for the Big Dipper (a saucepan of seven stars in the Great Bear). Draw a line through the two stars at the front of the pan and go up five times as far. That is the Pole Star. When you face it, you are facing north. All other stars slowly turn around it during the night.",
		}},
	{"milkyway", [2]string{"De Melkweg", "The Milky Way"},
		[2]string{"zo'n 100 tot 400 miljard sterren · 100.000 lichtjaar breed", "about 100 to 400 billion stars · 100,000 light years across"},
		[2]string{
			"Onze zon is één van de honderden miljarden sterren in een enorme draaiende schijf: de Melkweg. Wij zitten ergens aan de rand van die schijf. Als je op een heel donkere plek naar de hemel kijkt, zie je een vage witte band: dat is de schijf van opzij gezien, met zoveel sterren dat ze samen een melkachtige streep vormen. In de stad zie je hem door al het licht niet. Er zijn nog veel meer sterrenstelsels, en het dichtstbijzijnde grote, de Andromedanevel, kun je in de herfst als vaag vlekje zien.",
			"Our sun is one of hundreds of billions of stars in a huge spinning disc: the Milky Way. We live somewhere towards the edge of that disc. If you look at the sky from a really dark place, you see a faint white band: that is the disc seen from the side, with so many stars that together they look like a milky stripe. In a city the lights hide it. There are many more galaxies, and the nearest big one, the Andromeda Galaxy, can be seen in autumn as a faint smudge.",
		}},
	{"lightyear", [2]string{"Een lichtjaar", "A light year"},
		[2]string{"licht gaat 300.000 km per seconde · 1 lichtjaar is ongeveer 9,5 biljoen km", "light travels 300,000 km per second · 1 light year is about 9.5 trillion km"},
		[2]string{
			"Een lichtjaar klinkt als een tijd, maar het is een afstand: hoe ver het licht in één jaar komt. Licht is het snelste wat er is. In één seconde gaat het zeven keer om de aarde heen. Toch doet het licht van de dichtstbijzijnde ster er ruim vier jaar over om bij ons te komen. Dat betekent ook: als je naar een ster kijkt, zie je hem zoals hij was toen het licht vertrok. Naar de hemel kijken is dus een beetje terugkijken in de tijd.",
			"A light year sounds like a time, but it is a distance: how far light travels in one year. Light is the fastest thing there is. In one second it goes around Earth seven times. Even so, light from the nearest star takes over four years to reach us. That also means: when you look at a star, you see it as it was when the light left. Looking at the sky is a bit like looking back in time.",
		}},
	{"meteors", [2]string{"Vallende sterren", "Shooting stars"},
		[2]string{"Perseïden: rond 12 augustus · Geminiden: rond 14 december · tot 100 per uur op een donkere plek", "Perseids: around 12 August · Geminids: around 14 December · up to 100 an hour from a dark place"},
		[2]string{
			"Een vallende ster is helemaal geen ster. Het is een stofje of steentje uit de ruimte, vaak niet groter dan een zandkorrel, dat met enorme snelheid onze lucht in vliegt. Door de wrijving gaat het gloeien en verbrandt het, hoog boven je hoofd. Als de aarde door een stofspoor van een komeet vliegt, zie je er veel tegelijk: een meteorenzwerm. De mooiste zijn de Perseïden in augustus en de Geminiden in december. Ga liggen, kijk omhoog en geef je ogen een kwartier om aan het donker te wennen.",
			"A shooting star is not a star at all. It is a speck of dust or a tiny stone from space, often no bigger than a grain of sand, that flies into our air at huge speed. Friction makes it glow and burn up, high above your head. When Earth passes through a comet's dust trail, you see lots at once: a meteor shower. The best are the Perseids in August and the Geminids in December. Lie down, look up and give your eyes fifteen minutes to get used to the dark.",
		}},
	{"comets", [2]string{"Kometen", "Comets"},
		[2]string{"de komeet Halley komt om de 76 jaar terug, volgende keer in 2061", "Halley's Comet returns every 76 years, next time in 2061"},
		[2]string{
			"Een komeet is een vuile sneeuwbal van ijs, stof en steen, een paar kilometer groot. De meeste tijd zweven ze ver weg en koud door de ruimte. Komt een komeet dicht bij de zon, dan warmt hij op en gaat het ijs verdampen. Er ontstaat een wolk om de kern en een lange staart, die altijd van de zon af wijst. Soms is zo'n staart miljoenen kilometers lang. Een heldere komeet zie je maar een paar keer in je leven, dus als er een komt, is het de moeite waard om naar buiten te gaan.",
			"A comet is a dirty snowball of ice, dust and rock, a few kilometres across. Most of the time they drift far away and cold through space. When a comet comes close to the sun, it warms up and the ice starts to evaporate. A cloud forms around the core, plus a long tail that always points away from the sun. Sometimes that tail is millions of kilometres long. You only see a bright comet a few times in your life, so if one comes, it is worth going outside.",
		}},
	{"eclipses", [2]string{"Verduisteringen", "Eclipses"},
		[2]string{"volgende totale zonsverduistering in Europa: 2 augustus 2027, in het zuiden van Spanje", "next total solar eclipse in Europe: 2 August 2027, in southern Spain"},
		[2]string{
			"Bij een zonsverduistering schuift de maan precies tussen de zon en de aarde. De maan is 400 keer kleiner dan de zon, maar staat ook ongeveer 400 keer dichterbij, dus ze past er net voor. Bij een totale verduistering wordt het midden op de dag even schemerig en zie je de stralenkrans van de zon. Kijk er alleen naar met een speciale eclipsbril. Bij een maansverduistering staat de aarde tussen de zon en de maan. De maan kleurt dan roodbruin, en daar mag je gewoon naar kijken.",
			"In a solar eclipse the moon slides exactly between the sun and Earth. The moon is 400 times smaller than the sun but also about 400 times closer, so it just fits in front. In a total eclipse it gets dim in the middle of the day and you can see the sun's glowing crown. Only look at it through special eclipse glasses. In a lunar eclipse Earth is between the sun and the moon. The moon turns reddish brown, and that is perfectly safe to look at.",
		}},
	{"blackholes", [2]string{"Zwarte gaten", "Black holes"},
		[2]string{"in het midden van de Melkweg zit er een: Sagittarius A*, 4 miljoen keer zo zwaar als de zon", "there is one in the middle of the Milky Way: Sagittarius A*, 4 million times as heavy as the sun"},
		[2]string{
			"Een zwart gat is een plek waar zoveel massa in een heel klein stukje ruimte zit dat niets meer kan ontsnappen, zelfs licht niet. Daarom is het zwart. Zwarte gaten ontstaan bijvoorbeeld als een hele grote ster opbrandt en in elkaar stort. Je kunt ze niet direct zien, maar wel wat eromheen gebeurt: gas dat eromheen raast wordt gloeiend heet. In 2019 maakten sterrenkundigen de eerste foto van de schaduw van een zwart gat. Geen zorgen: er is er geen in de buurt van de aarde.",
			"A black hole is a place where so much mass is packed into such a tiny space that nothing can escape, not even light. That is why it is black. Black holes form, for example, when a very big star burns out and collapses. You cannot see them directly, but you can see what happens around them: gas racing around them gets glowing hot. In 2019 astronomers took the first picture of a black hole's shadow. Don't worry: there is none anywhere near Earth.",
		}},
	{"iss", [2]string{"Het ISS en astronauten", "The ISS and astronauts"},
		[2]string{"ongeveer 400 km hoog · 28.000 km per uur · 16 keer per dag rond de aarde", "about 400 km up · 28,000 km per hour · 16 times around Earth every day"},
		[2]string{
			"Het internationale ruimtestation is zo groot als een voetbalveld en er wonen bijna altijd zeven astronauten uit verschillende landen. Ze doen proeven die op aarde niet kunnen, bijvoorbeeld met planten en vuur zonder zwaartekracht. Omdat het station steeds om de aarde 'valt', zweeft alles erin. Astronauten slapen in een slaapzak aan de muur. Ze zien elke dag 16 zonsopkomsten. Je kunt het ISS zelf zien als het 's avonds nog in de zon vliegt terwijl het bij jou al donker is: kijk in het lijstje hiernaast wanneer.",
			"The International Space Station is as big as a football pitch and almost always has seven astronauts from different countries living on board. They do experiments that are impossible on Earth, for instance with plants and fire without gravity. Because the station keeps 'falling' around Earth, everything inside floats. Astronauts sleep in a sleeping bag strapped to the wall. They see 16 sunrises every day. You can see the ISS yourself when it is still in sunlight while it is already dark where you are: check the list for when.",
		}},
}

func (m model) taalIndex() int {
	if m.inst.Taal == "nl" {
		return 0
	}
	return 1
}

// viewHemelUitleg: links de onderwerpen, rechts het verhaal.
func (m model) viewHemelUitleg(p palet, kop string, breed int) string {
	h := m.od.hemel
	ti := m.taalIndex()
	dim := lipgloss.NewStyle().Foreground(p.dim)
	acc := lipgloss.NewStyle().Foreground(p.acc).Bold(true)
	links := min(30, breed/3)
	rechts := breed - links - 5

	zicht := max(m.hoog-9, 6)
	begin := 0
	if h.keuze >= zicht {
		begin = h.keuze - zicht + 1
	}
	var l []string
	for i := begin; i < len(hemelOnderwerpen) && i < begin+zicht; i++ {
		titel := hemelOnderwerpen[i].titel[ti]
		if i == h.keuze {
			l = append(l, acc.Render("› "+titel))
		} else {
			l = append(l, "  "+titel)
		}
	}

	o := hemelOnderwerpen[h.keuze]
	tekst := lipgloss.NewStyle().Width(rechts - 4)
	r := []string{acc.Render(o.titel[ti]), tekst.Render(dim.Render(o.feiten[ti])), "", tekst.Render(o.tekst[ti])}
	if o.sleutel == "moon" {
		nu := time.Now()
		_, verlicht := maanStand(nu)
		r = append(r, "", tekst.Render(maanTekens[maanFase(nu)]+" "+m.t("sky.moonnow", int(verlicht*100+0.5), dagenTotVolleMaan(nu))))
	}
	// lange tekst op een klein scherm: met spatie of pijl rechts verder lezen
	regels := strings.Split(strings.Join(r, "\n"), "\n")
	ruimte := max(zicht-1, 4)
	h.scroll = max(min(h.scroll, len(regels)-ruimte), 0)
	if len(regels) > ruimte {
		eind := min(h.scroll+ruimte, len(regels))
		meer := ""
		if eind < len(regels) {
			meer = dim.Render(m.t("sky.more"))
		}
		regels = append(regels[h.scroll:eind:eind], meer)
	}
	return kop + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top,
		paneel(p, m.t("sky.topics"), strings.Join(l, "\n"), links, true), " ",
		paneel(p, fmt.Sprintf("%d/%d", h.keuze+1, len(hemelOnderwerpen)), strings.Join(regels, "\n"), rechts, false)) +
		"\n" + dim.Render(m.t("sky.learnkeys"))
}
