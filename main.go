package main

import (
	"github.com/secnex/bin-api/server"
)

func main() {
	s := server.NewServer("0.0.0.0", 8081)
	s.Start()
}
