package main

import (
	"log"

	"github.com/stevedylandev/andromeda/apps/sipp-go/server"
	"github.com/stevedylandev/andromeda/crates-go/config"
)

func main() {
	config.LoadDotEnv(".env")
	host := config.Getenv("HOST", "127.0.0.1")
	port := config.GetenvInt("PORT", 3000)
	if err := server.Run(host, port); err != nil {
		log.Fatal(err)
	}
}
