// Command linkforged is the Linkforge service binary.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jh2wnemwx/rd-linkforge/internal/link"
	"github.com/jh2wnemwx/rd-linkforge/internal/store"
)

// version is overwritten at build time with -ldflags "-X main.version=...".
// var version = "dev"

func main() {
	args := os.Args[1:]
	s := store.New()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	for i, v := range args {
		if v == "" {
			continue
		}

		l, err := link.New(uint64(i), v)
		if err != nil {
			logger.Error("couldn't create link", "link", v, "err", err)
		}

		if err := store.Add(s, l); err != nil {
			logger.Error("couldn't add link to store", "link", l.Target, "store", s, "err", err)
		}
	}

	for _, v := range store.All(s) {
		fmt.Printf("%v\t%v\n", v.Code, v.Target)
	}
	fmt.Println("\nTotal:", store.Count(s))
	// fmt.Printf("linkforged %s\n", version)
}
