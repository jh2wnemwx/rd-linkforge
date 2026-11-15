// Command linkforged is the Linkforge service binary.
package main

import (
	"context"
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
	ctx := context.Background()
	owner := "demo"

	s := store.New()
	svc := link.NewService(s)

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	for _, v := range args {
		if v == "" {
			continue
		}

		req := link.ShortenRequest{
			Target:  v,
			OwnerID: owner,
		}

		_, err := svc.Shorten(ctx, req)
		if err != nil {
			logger.Error("couldn't shorten", "ShortenRequest", req, "err", err)
		}
	}

	links, err := svc.List(ctx, owner, link.Page{})
	if err != nil {
		logger.Error("couldn't list", "owner", owner, "err", err)
	}

	for _, v := range links {
		fmt.Printf("%v\t%v\n", v.Code, v.Target)
	}

	fmt.Println("\nTotal:", s.Count())
}
