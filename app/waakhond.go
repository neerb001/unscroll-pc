package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// De waakhond. De app krijgt elke 20 seconden een kloktik; elke toets en elke tik gaat door Update.
// Komt daar drie minuten lang niets meer door, dan zit de app vast (het scherm reageert niet,
// het avondslot gaat niet dicht). Dan schrijft de waakhond op waar het programma vastzat en
// stopt hij de app; unscroll-sessie start hem meteen opnieuw. De tijd loopt op de monotone klok,
// dus een computer die even sliep telt niet als vastgelopen.

const waakhondGrens = 3 * time.Minute

var (
	hartslagMu sync.Mutex
	hartslag   = time.Now()
)

// klop wordt aan het begin van elke Update aangeroepen.
func klop() {
	hartslagMu.Lock()
	hartslag = time.Now()
	hartslagMu.Unlock()
}

func stilSinds() time.Duration {
	hartslagMu.Lock()
	defer hartslagMu.Unlock()
	return time.Since(hartslag)
}

// vastgelopenPad: hier komt het verslag, zodat een ouder het via ssh kan bekijken.
func vastgelopenPad() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "unscroll", "vastgelopen.txt")
}

// schrijfVerslag zet de stand van alle onderdelen van het programma in een bestand.
func schrijfVerslag(stil time.Duration) string {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	pad := vastgelopenPad()
	_ = os.MkdirAll(filepath.Dir(pad), 0o755)
	kop := fmt.Sprintf("Unscroll %s zat vast: %s geen reactie, gestopt op %s\n\n", versie, stil.Round(time.Second), time.Now().Format("2006-01-02 15:04:05"))
	_ = veiligSchrijven(pad, append([]byte(kop), buf[:n]...), 0o644)
	return pad
}

// startWaakhond kijkt elke halve minuut. stop wordt aangeroepen als de app vastzit.
func startWaakhond(stop func()) {
	go func() {
		for range time.Tick(30 * time.Second) {
			if stil := stilSinds(); stil > waakhondGrens {
				schrijfVerslag(stil)
				stop()
				return
			}
		}
	}()
}
