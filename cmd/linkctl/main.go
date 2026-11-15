// Command linkctl is the Linkforge operator CLI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jh2wnemwx/rd-linkforge/internal/config"
	"github.com/jh2wnemwx/rd-linkforge/internal/link"
	"github.com/jh2wnemwx/rd-linkforge/internal/linkio"
	"github.com/jh2wnemwx/rd-linkforge/internal/store"
)

// version is overwritten at build time with -ldflags "-X main.version=...".
var version = "dev"

// Exit codes.
const (
	exitOK      = 0 // success
	exitFailure = 1 // the command ran and failed
	exitUsage   = 2 // the command was used wrongly
)

// importOwner is the owner of every link created by "linkctl import".
const importOwner = "cli"

const usage = `usage: linkctl <command> [flags]

commands:
  import  -in FILE -db FILE     shorten every target of FILE into the database
  export  -db FILE [-out FILE]  write the links of the database as CSV
  config                        load and check the configuration from the environment
  version                       print the version
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// run executes the command given by args and returns the process exit code.
func run(ctx context.Context, args []string, env func(string) (string, bool), stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, "linkctl: no command\n\n", usage)
		return exitUsage
	}

	cmd, args := args[0], args[1:]

	if cmd == "help" || cmd == "-h" || cmd == "-help" || cmd == "--help" {
		fmt.Fprint(stderr, usage)
		return exitOK
	}

	lvl, err := logLevel(env)
	if err != nil {
		fmt.Fprintf(stderr, "linkctl: %v\n", err)
		return exitUsage
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: lvl}))

	switch cmd {
	case "import":
		return runImport(ctx, args, logger, stdout, stderr)
	case "export":
		return runExport(args, stdout, stderr)
	case "config":
		return runConfig(ctx, args, env, logger, stdout, stderr)
	case "version":
		return runVersion(args, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "linkctl: unknown command %q\n\n%s", cmd, usage)
		return exitUsage
	}
}

// logLevel reads LF_LOG_LEVEL. An unset or blank value means info.
func logLevel(env func(string) (string, bool)) (slog.Level, error) {
	v, _ := env("LF_LOG_LEVEL")
	lvl, err := config.ParseLevel(v)
	if err != nil {
		return 0, fmt.Errorf("LF_LOG_LEVEL %q: %w", v, err)
	}
	return lvl, nil
}

// newFlagSet returns a flag set that reports errors instead of exiting and
// writes its messages to stderr.
func newFlagSet(name, synopsis string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(stderr, "usage: linkctl %s %s\n", name, synopsis)
		flags.PrintDefaults()
	}
	return flags
}

// parseFlags parses args and rejects positional arguments. When ok is false,
// code is the exit code to return: 0 for -h, 2 for any misuse.
func parseFlags(flags *flag.FlagSet, args []string, stderr io.Writer) (code int, ok bool) {
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK, false
		}
		return exitUsage, false // the flag package already printed the error and usage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "linkctl %s: unexpected argument %q\n", flags.Name(), flags.Arg(0))
		flags.Usage()
		return exitUsage, false
	}
	return exitOK, true
}

// requireFlags reports the first of names whose value is empty.
func requireFlags(flags *flag.FlagSet, stderr io.Writer, values map[string]string, names ...string) bool {
	for _, name := range names {
		if values[name] == "" {
			fmt.Fprintf(stderr, "linkctl %s: flag -%s is required\n", flags.Name(), name)
			flags.Usage()
			return false
		}
	}
	return true
}

func runImport(ctx context.Context, args []string, logger *slog.Logger, stdout, stderr io.Writer) int {
	flags := newFlagSet("import", "-in FILE -db FILE", stderr)
	in := flags.String("in", "", "text file with one target URL per line (required)")
	db := flags.String("db", "", "database file (JSON Lines) to add the links to (required)")
	if code, ok := parseFlags(flags, args, stderr); !ok {
		return code
	}
	if !requireFlags(flags, stderr, map[string]string{"in": *in, "db": *db}, "in", "db") {
		return exitUsage
	}

	fail := func(err error) int {
		fmt.Fprintf(stderr, "linkctl import: %v\n", err)
		return exitFailure
	}

	targets, err := readTargetsFile(*in)
	if err != nil {
		return fail(err)
	}

	existing, err := linkio.LoadFile(*db)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fail(err)
	}

	st := store.New()
	var maxID uint64
	for _, l := range existing {
		if _, err := st.Create(ctx, l); err != nil {
			return fail(fmt.Errorf("database link %d (%s): %w", l.ID, l.Code, err))
		}
		maxID = max(maxID, l.ID)
	}

	svc := link.NewService(st, link.WithFirstID(maxID+1))

	imported, skipped := 0, 0
	for _, t := range targets {
		if ctx.Err() != nil {
			break
		}
		_, err := svc.Shorten(ctx, link.ShortenRequest{Target: t.URL, OwnerID: importOwner})
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			logger.Warn("skipped target", "line", t.Line, "target", t.URL, "err", err)
			skipped++
			continue
		}
		imported++
	}

	// A cancelled import must leave the database file untouched.
	if err := ctx.Err(); err != nil {
		return fail(fmt.Errorf("interrupted, database not changed: %w", err))
	}

	if err := linkio.SaveFile(*db, st.All()); err != nil {
		return fail(err)
	}

	fmt.Fprintf(stdout, "imported: %d\nskipped: %d\n", imported, skipped)
	if skipped > 0 {
		return exitFailure
	}
	return exitOK
}

func readTargetsFile(path string) ([]linkio.Target, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() // read-only: nothing to lose on Close

	return linkio.ReadTargets(f)
}

func runExport(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("export", "-db FILE [-out FILE]", stderr)
	db := flags.String("db", "", "database file (JSON Lines) to read (required)")
	out := flags.String("out", "", "CSV file to write (default: stdout)")
	if code, ok := parseFlags(flags, args, stderr); !ok {
		return code
	}
	if !requireFlags(flags, stderr, map[string]string{"db": *db}, "db") {
		return exitUsage
	}

	fail := func(err error) int {
		fmt.Fprintf(stderr, "linkctl export: %v\n", err)
		return exitFailure
	}

	links, err := linkio.LoadFile(*db)
	if errors.Is(err, fs.ErrNotExist) {
		return fail(fmt.Errorf("database file %s does not exist", *db))
	}
	if err != nil {
		return fail(err)
	}

	if *out == "" {
		if err := linkio.WriteCSV(stdout, links); err != nil {
			return fail(err)
		}
		return exitOK
	}

	if err := writeCSVFile(*out, links); err != nil {
		return fail(err)
	}
	return exitOK
}

// writeCSVFile writes links as CSV to path. The error from Close is reported.
func writeCSVFile(path string, links []link.Link) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = linkio.WriteCSV(f, links)
	if err = errors.Join(err, f.Close()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func runConfig(ctx context.Context, args []string, env func(string) (string, bool), logger *slog.Logger, stdout, stderr io.Writer) int {
	flags := newFlagSet("config", "", stderr)
	if code, ok := parseFlags(flags, args, stderr); !ok {
		return code
	}

	cfg, err := config.NewEnvLoader(env).Load(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "invalid configuration: %v\n", err)
		return exitFailure
	}

	// Config implements slog.LogValuer, so the credentials are redacted.
	logger.Info("configuration loaded", "config", cfg)
	fmt.Fprintln(stdout, "config ok")
	return exitOK
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("version", "", stderr)
	if code, ok := parseFlags(flags, args, stderr); !ok {
		return code
	}
	fmt.Fprintf(stdout, "linkctl %s\n", version)
	return exitOK
}
