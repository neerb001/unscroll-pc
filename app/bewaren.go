package main

import (
	"os"
	"path/filepath"
	"sync"
)

// Veilig bewaren. De Eee PC heeft geen werkende accu: als de stekker eruit gaat, is hij meteen uit.
// Een bestand dat op dat moment half geschreven is, kan dan leeg of kapot zijn. Daarom schrijven
// we nooit over een bestaand bestand heen, maar:
//  1. alles naar een nieuw tijdelijk bestand in dezelfde map,
//  2. dat in één keer op de plek van het oude zetten (rename: dat is óf helemaal gebeurd, óf niet),
//  3. het bestand en de map echt naar de schijf laten schrijven (fsync).
// Na een stroomstoring heb je dus altijd óf de oude, óf de nieuwe versie, nooit een half bestand.
//
// Stap 3 kan op een oude, trage schijf zomaar een halve seconde duren. In de hoofdlus van de app
// zou elke schaakzet of elk sudokucijfer dan haperen. Daarom doet veiligSchrijven stap 3 op de
// achtergrond: het bestandssysteem (ext4) schrijft een bestand dat via rename een ander vervangt
// zelf al vóór de naamswissel weg, dus ook in die korte tussentijd blijft het oude of het nieuwe
// bestand heel. veiligSchrijvenNu wacht wel op de schijf: voor ouders.yaml, dat door een kort
// hulpprogramma wordt geschreven.

func veiligSchrijven(pad string, data []byte, perm os.FileMode) error {
	if err := schrijfEnWissel(pad, data, perm, false); err != nil {
		return err
	}
	syncStraks(pad)
	return nil
}

func veiligSchrijvenNu(pad string, data []byte, perm os.FileMode) error {
	if err := schrijfEnWissel(pad, data, perm, true); err != nil {
		return err
	}
	syncMap(filepath.Dir(pad))
	return nil
}

func schrijfEnWissel(pad string, data []byte, perm os.FileMode, metSync bool) error {
	dir := filepath.Dir(pad)
	f, err := os.CreateTemp(dir, "."+filepath.Base(pad)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	mislukt := func(err error) error {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if _, err := f.Write(data); err != nil {
		return mislukt(err)
	}
	if err := f.Chmod(perm); err != nil {
		return mislukt(err)
	}
	if metSync {
		if err := f.Sync(); err != nil {
			return mislukt(err)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, pad); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// De wachtrij voor de schijf: bestanden die nog naar de schijf gedwongen moeten worden.
var (
	syncMu      sync.Mutex
	syncWacht   = map[string]bool{}
	syncBezig   sync.WaitGroup
	syncSein    = make(chan struct{}, 1)
	syncGestart sync.Once
)

func syncStraks(pad string) {
	syncGestart.Do(func() { go syncWerker() })
	syncMu.Lock()
	if !syncWacht[pad] {
		syncWacht[pad] = true
		syncBezig.Add(1)
	}
	syncMu.Unlock()
	select {
	case syncSein <- struct{}{}:
	default:
	}
}

func syncWerker() {
	for range syncSein {
		for {
			syncMu.Lock()
			var pad string
			for p := range syncWacht {
				pad = p
				break
			}
			if pad == "" {
				syncMu.Unlock()
				break
			}
			delete(syncWacht, pad)
			syncMu.Unlock()
			if f, err := os.Open(pad); err == nil {
				_ = f.Sync()
				f.Close()
			}
			syncMap(filepath.Dir(pad))
			syncBezig.Done()
		}
	}
}

// wachtOpSchijf wacht tot alles wat bewaard is ook echt op de schijf staat (bij afsluiten).
func wachtOpSchijf() { syncBezig.Wait() }

// syncMap laat de map zelf naar de schijf schrijven (de nieuwe bestandsnaam).
func syncMap(dir string) {
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
}

// veiligToevoegen zet een regel achter aan een bestand (dagboek, verhaal) en laat hem meteen
// naar de schijf schrijven. Bij een stroomstoring kan hooguit de allerlaatste regel missen.
func veiligToevoegen(pad string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(pad, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
