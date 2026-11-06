package main

import (
	"log"
	"os"
	"strings"

	"envman/internal/app"
)

func main() {
	var dataDir string
	for _, argument := range os.Args[1:] {
		if value, ok := strings.CutPrefix(argument, "--data-dir="); ok {
			dataDir = strings.Trim(value, `"`)
		}
	}
	if err := app.Run(dataDir); err != nil {
		log.Fatal(err)
	}
}
