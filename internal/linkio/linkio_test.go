package linkio

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jh2wnemwx/rd-linkforge/internal/link"
)

var (
	created = time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	expires = time.Date(2030, 2, 3, 4, 5, 6, 0, time.UTC)
)

func sampleLinks() []link.Link {
	return []link.Link{
		{ID: 1, Code: "000001", Target: "https://example.com", OwnerID: "u1", CreatedAt: created},
		{ID: 2, Code: "my-link", Target: "https://go.dev/?a=1,b=2", OwnerID: "u2", CreatedAt: created, ExpiresAt: expires},
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestReadTargets(t *testing.T) {
	in := "https://a.io\n\n   \n# comment\n  https://b.io  \n\t#indented comment\nhttps://c.io"
	got, err := ReadTargets(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []Target{{1, "https://a.io"}, {5, "https://b.io"}, {7, "https://c.io"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestReadTargetsReaderError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := ReadTargets(errReader{boom}); !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap %v", err, boom)
	}
}

func TestLinksRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteLinks(&buf, sampleLinks()); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(buf.String(), "\n"); n != 2 {
		t.Errorf("wrote %d lines, want 2:\n%s", n, buf.String())
	}
	got, err := ReadLinks(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sampleLinks()) {
		t.Errorf("got %+v, want %+v", got, sampleLinks())
	}
}

func TestReadLinksEmpty(t *testing.T) {
	got, err := ReadLinks(strings.NewReader(""))
	if err != nil || len(got) != 0 {
		t.Errorf("got %v, %v; want no links", got, err)
	}
}

func TestReadLinksBadRecord(t *testing.T) {
	in := `{"ID":1}` + "\n" + `{"ID":2}` + "\n" + `{"ID":"x"}` + "\n"
	_, err := ReadLinks(strings.NewReader(in))
	if err == nil || !strings.Contains(err.Error(), "record 3:") {
		t.Errorf("err = %v, want it to name record 3", err)
	}
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, sampleLinks()); err != nil {
		t.Fatal(err)
	}
	want := "id,code,target,owner_id,created_at,expires_at\n" +
		"1,000001,https://example.com,u1,2030-01-02T03:04:05Z,\n" +
		`2,my-link,"https://go.dev/?a=1,b=2",u2,2030-01-02T03:04:05Z,2030-02-03T04:05:06Z` + "\n"
	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestWriteCSVConvertsToUTC(t *testing.T) {
	zone := time.FixedZone("X", 3*3600)
	var buf bytes.Buffer
	l := link.Link{ID: 1, CreatedAt: created.In(zone)}
	if err := WriteCSV(&buf, []link.Link{l}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "2030-01-02T03:04:05Z") {
		t.Errorf("created_at not in UTC:\n%s", buf.String())
	}
}

func TestSaveAndLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.jsonl")
	if err := SaveFile(path, sampleLinks()); err != nil {
		t.Fatal(err)
	}
	// Saving again replaces the file rather than appending to it.
	if err := SaveFile(path, sampleLinks()[:1]); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sampleLinks()[:1]) {
		t.Errorf("got %+v, want %+v", got, sampleLinks()[:1])
	}
}

func TestLoadFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.jsonl")
	_, err := LoadFile(path)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want fs.ErrNotExist", err)
	}
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("err = %v, want it to contain the path", err)
	}
}

func TestLoadFileBadRecordNamesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte("{\"ID\":1}\nnot json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(path)
	if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "record 2:") {
		t.Errorf("err = %v, want path and record 2", err)
	}
}

func TestSaveFileErrorNamesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "links.jsonl")
	err := SaveFile(path, sampleLinks())
	if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), path) {
		t.Errorf("err = %v, want fs.ErrNotExist naming the path", err)
	}
}
