package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/stockyard-dev/stockyard-sundial/internal/server"
	"github.com/stockyard-dev/stockyard-sundial/internal/store"
	"github.com/stockyard-dev/stockyard/bus"
)

var version = "dev"

func main() {
	portFlag := flag.String("port", "", "HTTP port (overrides PORT env var)")
	dataFlag := flag.String("data", "", "Data directory (overrides DATA_DIR env var)")
	flag.Parse()

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "9700"
	}

	dataDir := *dataFlag
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "./sundial-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("sundial: %v", err)
	}
	defer db.Close()

	// Bus: one level up from the private data dir so every tool in a
	// bundle shares one _bus.db. Non-fatal: sundial serves users with
	// or without it.
	var b *bus.Bus
	if bb, berr := bus.Open(filepath.Dir(dataDir), "sundial"); berr != nil {
		log.Printf("sundial: bus disabled: %v", berr)
	} else {
		b = bb
		defer b.Close()
	}

	srv := server.New(db, server.DefaultLimits(dataDir), dataDir, b)

	fmt.Printf("\n  Sundial v%s — Self-hosted time tracker\n", version)
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  Questions?  hello@stockyard.dev — I read every message\n\n")

	log.Printf("sundial: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
