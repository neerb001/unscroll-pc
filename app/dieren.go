package main

// De dieren van "Dier van de dag": Engelse Wikipedia-titels. De Nederlandse pagina zoekt de app
// op via de taalkoppeling van Wikipedia. De lijst staat in de app: Wikipedia kiest niets zelf.
var dieren = []string{
	// zoogdieren
	"Hedgehog", "Red fox", "European badger", "Eurasian beaver", "Red squirrel", "Eurasian otter", "Sea otter",
	"Roe deer", "Red deer", "Reindeer", "Moose", "Wild boar", "European hare", "European rabbit", "Mole (animal)",
	"Bat", "Pipistrellus", "Wolf", "Eurasian lynx", "Brown bear", "Polar bear", "Giant panda", "Red panda",
	"African bush elephant", "Asian elephant", "Giraffe", "Okapi", "Plains zebra", "Hippopotamus", "White rhinoceros",
	"Lion", "Tiger", "Snow leopard", "Cheetah", "Jaguar", "Leopard", "Caracal", "Serval", "Meerkat", "Aardvark",
	"Pangolin", "Armadillo", "Sloth", "Anteater", "Tapir", "Capybara", "Chinchilla", "Porcupine", "Kangaroo",
	"Koala", "Wombat", "Platypus", "Echidna", "Tasmanian devil", "Sugar glider", "Chimpanzee", "Bonobo", "Gorilla",
	"Orangutan", "Gibbon", "Ring-tailed lemur", "Aye-aye", "Slow loris", "Tarsier", "Hamadryas baboon", "Mandrill",
	"Japanese macaque", "Camel", "Llama", "Alpaca", "American bison", "Musk ox", "Mountain goat", "Alpine ibex",
	"Fennec fox", "Arctic fox", "Raccoon", "Skunk", "Wolverine", "Honey badger", "Stoat", "Least weasel",
	"European polecat", "Pine marten", "Harbor seal", "Grey seal", "Walrus", "Sea lion", "Manatee", "Dugong",
	"Humpback whale", "Blue whale", "Sperm whale", "Orca", "Narwhal", "Beluga whale", "Bottlenose dolphin",
	"Harbour porpoise", "Naked mole-rat", "Dormouse", "Harvest mouse", "Hamster", "Guinea pig", "Prairie dog",
	"Groundhog", "Flying squirrel", "Shrew", "Elephant shrew",
	// vogels
	"Little owl", "Barn owl", "Eurasian eagle-owl", "Snowy owl", "Tawny owl", "Common kingfisher", "Great tit",
	"Eurasian blue tit", "European robin", "Common blackbird", "House sparrow", "Common starling", "Common swift",
	"Barn swallow", "White stork", "Grey heron", "Common crane", "Eurasian spoonbill", "Pied avocet",
	"Eurasian oystercatcher", "Great crested grebe", "Mute swan", "Mallard", "Barnacle goose", "Common cuckoo",
	"Great spotted woodpecker", "Eurasian jay", "Common raven", "Hooded crow", "Eurasian magpie", "Common buzzard",
	"Peregrine falcon", "Common kestrel", "Golden eagle", "Bald eagle", "Red kite", "Osprey", "Bearded vulture",
	"Andean condor", "Atlantic puffin", "Northern gannet", "Emperor penguin", "King penguin", "Little penguin",
	"Ostrich", "Emu", "Cassowary", "Kiwi (bird)", "Kakapo", "Flamingo", "Pelican", "Toucan", "Scarlet macaw",
	"Cockatoo", "Budgerigar", "Hummingbird", "Indian peafowl", "Lyrebird", "Bowerbird", "Shoebill", "Hoatzin",
	"Wandering albatross", "Arctic tern", "Woodpecker finch", "Common nightingale",
	// reptielen en amfibieën
	"Common toad", "Common frog", "Poison dart frog", "Red-eyed tree frog", "Axolotl", "Fire salamander",
	"Great crested newt", "Chameleon", "Gecko", "Pogona", "Green iguana", "Marine iguana", "Komodo dragon",
	"Gila monster", "Slow worm", "Sand lizard", "Grass snake", "European adder", "King cobra", "Reticulated python",
	"Green anaconda", "Saltwater crocodile", "American alligator", "Gharial", "Galápagos tortoise",
	"Leatherback sea turtle", "Green sea turtle", "Alligator snapping turtle", "Tuatara",
	// vissen en zeedieren
	"Seahorse", "Clownfish", "Great white shark", "Whale shark", "Hammerhead shark", "Manta ray", "Ocean sunfish",
	"Pufferfish", "Anglerfish", "Atlantic salmon", "Atlantic herring", "Northern pike", "Electric eel", "Piranha",
	"Flying fish", "Coelacanth", "Lionfish", "Octopus", "Common cuttlefish", "Giant squid", "Nautilus", "Jellyfish",
	"Portuguese man o' war", "Sea anemone", "Coral", "Starfish", "Sea urchin", "Sea cucumber", "Hermit crab",
	"European lobster", "Mantis shrimp", "Horseshoe crab", "Krill", "Tardigrade",
	// insecten en kleine beestjes
	"Western honey bee", "Bumblebee", "Ant", "Leafcutter ant", "Termite", "Coccinellidae", "Stag beetle", "Dung beetle",
	"Firefly", "Dragonfly", "Damselfly", "Mayfly", "Grasshopper", "Praying mantis", "Stick insect", "Monarch butterfly",
	"Peacock butterfly", "Atlas moth", "Silkworm", "Cicada", "Water strider", "Earwig", "Earthworm", "Snail",
	"Cornu aspersum", "Slug", "Woodlouse", "Millipede", "Centipede", "European garden spider", "Tarantula",
	"Jumping spider", "Scorpion", "Firebrat",
}
