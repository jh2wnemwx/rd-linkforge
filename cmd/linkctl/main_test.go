package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jh2wnemwx/rd-linkforge/internal/link"
	"github.com/jh2wnemwx/rd-linkforge/internal/linkio"
)

// result is what one call of run produced.
type result struct {
	code           int
	stdout, stderr string
}

// runWith calls run with a fixed environment, so the tests never see the
// real one.
func runWith(t *testing.T, ctx context.Context, env map[string]string, args ...string) result {
	t.Helper()
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
	var stdout, stderr bytes.Buffer
	code := run(ctx, args, lookup, &stdout, &stderr)
	return result{code, stdout.String(), stderr.String()}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func loadLinks(t *testing.T, path string) []link.Link {
	t.Helper()
	links, err := linkio.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return links
}

// paths returns an input file and a database file inside a fresh t.TempDir().
func paths(t *testing.T) (in, db string) {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "targets.txt"), filepath.Join(dir, "links.jsonl")
}

func importTargets(t *testing.T, in, db, targets string) result {
	t.Helper()
	writeFile(t, in, targets)
	return runWith(t, context.Background(), nil, "import", "-in", in, "-db", db)
}

func TestImportThenExport(t *testing.T) {
	in, db := paths(t)

	r := importTargets(t, in, db, "# targets\nhttps://a.example\n\n  https://b.example  \n")
	if r.code != exitOK || r.stdout != "imported: 2\nskipped: 0\n" {
		t.Fatalf("import: %+v", r)
	}

	r = runWith(t, context.Background(), nil, "export", "-db", db)
	if r.code != exitOK {
		t.Fatalf("export: %+v", r)
	}
	rows, err := csv.NewReader(strings.NewReader(r.stdout)).ReadAll()
	if err != nil {
		t.Fatalf("export is not CSV: %v\n%s", err, r.stdout)
	}
	want := [][]string{
		{"id", "code", "target", "owner_id"},
		{"1", "000001", "https://a.example", "cli"},
		{"2", "000002", "https://b.example", "cli"},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d CSV rows, want %d:\n%s", len(rows), len(want), r.stdout)
	}
	for i, w := range want {
		if got := rows[i][:4]; strings.Join(got, ",") != strings.Join(w, ",") {
			t.Errorf("row %d = %v, want %v", i, got, w)
		}
	}
}

func TestExportToFile(t *testing.T) {
	in, db := paths(t)
	if r := importTargets(t, in, db, "https://a.example\n"); r.code != exitOK {
		t.Fatalf("import: %+v", r)
	}

	out := filepath.Join(filepath.Dir(db), "links.csv")
	r := runWith(t, context.Background(), nil, "export", "-db", db, "-out", out)
	if r.code != exitOK || r.stdout != "" {
		t.Fatalf("export: %+v", r)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "id,code,target,owner_id,created_at,expires_at\n1,000001,https://a.example,cli,") {
		t.Errorf("CSV file:\n%s", b)
	}
}

func TestSecondImportContinuesNumbering(t *testing.T) {
	in, db := paths(t)
	if r := importTargets(t, in, db, "https://a.example\nhttps://b.example\n"); r.code != exitOK {
		t.Fatalf("first import: %+v", r)
	}
	if r := importTargets(t, in, db, "https://c.example\n"); r.code != exitOK || r.stdout != "imported: 1\nskipped: 0\n" {
		t.Fatalf("second import: %+v", r)
	}

	links := loadLinks(t, db)
	if len(links) != 3 {
		t.Fatalf("got %d links, want 3", len(links))
	}
	for i, l := range links {
		if l.ID != uint64(i+1) {
			t.Errorf("link %d has ID %d, want %d", i, l.ID, i+1)
		}
	}
	if links[2].Target != "https://c.example" || links[2].Code != "000003" {
		t.Errorf("third link = %+v", links[2])
	}
}

func TestImportSkipsBadLine(t *testing.T) {
	in, db := paths(t)

	r := importTargets(t, in, db, "https://a.example\nftp://bad.example\nhttps://c.example\n")
	if r.code != exitFailure {
		t.Errorf("code = %d, want %d", r.code, exitFailure)
	}
	if r.stdout != "imported: 2\nskipped: 1\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
	if !strings.Contains(r.stderr, "line=2") || !strings.Contains(r.stderr, "ftp://bad.example") {
		t.Errorf("stderr does not name the skipped line: %q", r.stderr)
	}

	links := loadLinks(t, db)
	if len(links) != 2 || links[0].Target != "https://a.example" || links[1].Target != "https://c.example" {
		t.Errorf("saved links = %+v, want the two good targets", links)
	}
}

func TestImportCancelledLeavesDatabaseUntouched(t *testing.T) {
	in, db := paths(t)
	if r := importTargets(t, in, db, "https://a.example\n"); r.code != exitOK {
		t.Fatalf("first import: %+v", r)
	}
	before, err := os.ReadFile(db)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, in, "https://b.example\nhttps://c.example\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := runWith(t, ctx, nil, "import", "-in", in, "-db", db)
	if r.code != exitFailure || r.stdout != "" {
		t.Errorf("cancelled import: %+v", r)
	}

	after, err := os.ReadFile(db)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("database changed:\nbefore %s\nafter  %s", before, after)
	}
}

func TestImportFailures(t *testing.T) {
	t.Run("missing input file", func(t *testing.T) {
		in, db := paths(t)
		r := runWith(t, context.Background(), nil, "import", "-in", in, "-db", db)
		if r.code != exitFailure || r.stderr == "" {
			t.Errorf("%+v", r)
		}
	})
	t.Run("corrupt database", func(t *testing.T) {
		in, db := paths(t)
		writeFile(t, db, "not json\n")
		r := importTargets(t, in, db, "https://a.example\n")
		if r.code != exitFailure || !strings.Contains(r.stderr, "record 1") {
			t.Errorf("%+v", r)
		}
	})
}

func TestExportMissingDatabase(t *testing.T) {
	_, db := paths(t)
	r := runWith(t, context.Background(), nil, "export", "-db", db)
	if r.code != exitFailure || r.stdout != "" || !strings.Contains(r.stderr, "does not exist") {
		t.Errorf("%+v", r)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{"no command", nil, nil},
		{"unknown command", nil, []string{"frobnicate"}},
		{"import unknown flag", nil, []string{"import", "-in", "t.txt", "-db", "db.jsonl", "-x"}},
		{"import missing -in", nil, []string{"import", "-db", "db.jsonl"}},
		{"import missing -db", nil, []string{"import", "-in", "t.txt"}},
		{"import flag without value", nil, []string{"import", "-in"}},
		{"export unknown flag", nil, []string{"export", "-db", "db.jsonl", "-x"}},
		{"export missing -db", nil, []string{"export"}},
		{"config unknown flag", nil, []string{"config", "-x"}},
		{"version unknown flag", nil, []string{"version", "-x"}},
		{"unexpected argument", nil, []string{"version", "extra"}},
		{"bad LF_LOG_LEVEL", map[string]string{"LF_LOG_LEVEL": "loud"}, []string{"version"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runWith(t, context.Background(), tt.env, tt.args...)
			if r.code != exitUsage {
				t.Errorf("code = %d, want %d", r.code, exitUsage)
			}
			if r.stdout != "" {
				t.Errorf("stdout = %q, want empty", r.stdout)
			}
			if r.stderr == "" {
				t.Error("stderr is empty, want a message")
			}
		})
	}
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"-h"}, {"import", "-h"}} {
		r := runWith(t, context.Background(), nil, args...)
		if r.code != exitOK || r.stdout != "" || !strings.Contains(r.stderr, "usage:") {
			t.Errorf("%v: %+v", args, r)
		}
	}
}

func TestVersion(t *testing.T) {
	r := runWith(t, context.Background(), nil, "version")
	if r.code != exitOK || r.stdout != "linkctl "+version+"\n" {
		t.Errorf("%+v", r)
	}
}

func TestConfigCommand(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"

	t.Run("good environment", func(t *testing.T) {
		env := map[string]string{
			"LF_PG_DSN":     "postgres://lf:pg-password@db/lf",
			"LF_MONGO_URI":  "mongodb://lf:mongo-password@mongo",
			"LF_JWT_SECRET": secret,
		}
		r := runWith(t, context.Background(), env, "config")
		if r.code != exitOK || r.stdout != "config ok\n" {
			t.Fatalf("%+v", r)
		}
		if !strings.Contains(r.stderr, "level=INFO") || !strings.Contains(r.stderr, "[redacted]") {
			t.Errorf("stderr does not log the redacted config: %s", r.stderr)
		}
		for _, s := range []string{"pg-password", "mongo-password", secret} {
			if strings.Contains(r.stderr, s) {
				t.Errorf("log leaks %q: %s", s, r.stderr)
			}
		}
	})

	t.Run("bad environment", func(t *testing.T) {
		env := map[string]string{"LF_CACHE_SIZE": "0"}
		r := runWith(t, context.Background(), env, "config")
		if r.code != exitFailure || r.stdout != "" {
			t.Fatalf("%+v", r)
		}
		if !strings.HasPrefix(r.stderr, "invalid configuration:") {
			t.Errorf("stderr = %q", r.stderr)
		}
		for _, key := range []string{"LF_PG_DSN", "LF_MONGO_URI", "LF_JWT_SECRET", "LF_CACHE_SIZE"} {
			if !strings.Contains(r.stderr, key) {
				t.Errorf("stderr does not list %s: %s", key, r.stderr)
			}
		}
	})
}
