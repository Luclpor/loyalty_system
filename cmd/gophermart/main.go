package main

import (
	"log"

	"github.com/Luclpor/loyalty_system.git/internal/server"
)

func main() {
	s, err := server.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	err = s.Start()
	if err != nil {
		log.Fatal(err)
	}
}
