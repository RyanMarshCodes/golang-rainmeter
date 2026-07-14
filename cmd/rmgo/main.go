package main

import (
	"log"
	"os"

	rmapp "github.com/RyanMarshCodes/golang-rainmeter/internal/app"
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
