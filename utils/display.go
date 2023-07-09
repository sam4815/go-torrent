package utils

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

type Display struct {
	Download *Download
	Quit     chan struct{}
}

func AnsiCleanUp() {
	fmt.Printf("\033[?25h")
}

func FlagUnicode(countryCode string) string {
	unicodeStart := 127462
	runeStart := 65

	flagUnicode := ""
	for _, letter := range countryCode {
		unicode := unicodeStart + (int(rune(letter)) - runeStart)
		flagUnicode += string(unicode)
	}

	return flagUnicode
}

func StartDisplay(download *Download, timeout time.Duration) Display {
	display := Display{Download: download, Quit: make(chan struct{})}
	ticker := time.NewTicker(timeout)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		<-signalChan
		AnsiCleanUp()
		os.Exit(1)
	}()

	// Hide cursor, create space, and position cursor in the middle
	fmt.Printf("\033[?25l\n\n")

	go func() {
		for {
			select {
			case <-ticker.C:
				if display.Download == nil {
					continue
				}

				display.Print()
			case <-display.Quit:
				return
			}
		}
	}()

	return display
}

func (display Display) Print() {
	fmt.Printf("\033[F%s %s\n", ProgressBar(display.Download), Countries(display.Download))
}

func (display Display) Close() {
	AnsiCleanUp()
	display.Print()
	fmt.Printf("\n")
	close(display.Quit)
}

func Countries(download *Download) string {
	emojis := make([]string, 0)

	for _, country := range download.ConnectedCountries {
		emojis = append(emojis, FlagUnicode(country))
	}

	return strings.Join(emojis, "  ")
}

func ProgressBar(download *Download) string {
	percentComplete := float64(len(download.CompletedPieceHash)) / float64(len(download.Torrent.PieceHash))

	progressBarSize := 50
	numCompletedBlocks := int(percentComplete * float64(progressBarSize))
	numEmptyBlocks := progressBarSize - numCompletedBlocks

	completedBlocks := strings.Repeat("█", numCompletedBlocks)
	emptyBlocks := strings.Repeat("░", numEmptyBlocks)

	return fmt.Sprintf(`%s%s %05.2f`, completedBlocks, emptyBlocks, percentComplete*100)
}
