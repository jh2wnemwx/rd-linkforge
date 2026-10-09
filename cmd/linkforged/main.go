// Command linkforged is the Linkforge service binary.
package main

import (
	"fmt"
	"github.com/jh2wnemwx/rd-linkforge/internal/link"
	"github.com/jh2wnemwx/rd-linkforge/internal/store"
	"os"
)

// version is overwritten at build time with -ldflags "-X main.version=...".
// var version = "dev"

func main() {
	args := os.Args[1:]
	s := store.New()

	for i, v := range args {
		if v == "" {
			continue
		}

		l, err := link.New(uint64(i), v)
		if err != nil {
			fmt.Printf("couldn't create link %q: %v", v, err)
		}

		if err := store.Add(s, l); err != nil {
			fmt.Printf("couldn't add link %q to store: %v", l, err)
		}
	}

	for _, v := range store.All(s) {
		fmt.Printf("%v\t%v\n", v.Code, v.Target)
	}
	fmt.Println("\nTotal:", store.Count(s))
	// fmt.Printf("linkforged %s\n", version)
}
