package main

import (
	"log"
	"os"

	"github.com/evanphx/columbia"
)

func main() {
	load := columbia.NewLoader()

	mod, err := load.LoadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	err = mod.Run([]string{"sh", "-c", `echo "START: $(date)"`})
	if err != nil {
		log.Fatal(err)
	}
}
