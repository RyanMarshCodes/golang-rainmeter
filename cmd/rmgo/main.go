package main

import (
	"log"
	"os"

	rmapp "ryanmarsh.net/rmgo/internal/app"
)

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := rmapp.Run(path); err != nil {
		log.Fatal(err)
	}
}
