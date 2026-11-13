package main

import (
	"os"
	"path/filepath"
)

// Veilig bewaren. De Eee PC heeft geen werkende accu: als de stekker eruit gaat, is hij meteen uit.
// Een bestand dat op dat moment half geschreven is, kan dan leeg of kapot zijn. Daarom schrijven
// we nooit over een bestaand bestand heen, maar:
//  1. alles naar een nieuw tijdelijk bestand in dezelfde map,
//  2. dat bestand echt naar de schijf laten schrijven (fsync),
//  3. het in één keer op de plek van het oude zetten (rename: dat is óf helemaal gebeurd, óf niet),
//  4. ook de map naar de schijf laten schrijven, zodat de nieuwe naam bewaard blijft.
// Na een stroomstoring heb je dus altijd óf de oude, óf de nieuwe versie, nooit een half bestand.

func veiligSchrijven(pad string, data []byte, perm os.FileMode) error {
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
	if err := f.Sync(); err != nil {
		return mislukt(err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, pad); err != nil {
		os.Remove(tmp)
		return err
	}
	syncMap(dir)
	return nil
}

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
