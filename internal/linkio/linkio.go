// Package linkio moves links in and out of files. It depends on link only and
// must not import store or config.
package linkio

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jh2wnemwx/rd-linkforge/internal/link"
)

// Target is one line of an import file.
type Target struct {
	Line int // 1-based line number in the file
	URL  string
}

// ReadTargets returns every line that is not blank and does not start with '#',
// with surrounding spaces trimmed. A reader error is returned wrapped.
func ReadTargets(r io.Reader) ([]Target, error) {
	var targets []Target

	sc := bufio.NewScanner(r)

	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		targets = append(targets, Target{line, text})
	}

	if err := sc.Err(); err != nil {
		return []Target{}, fmt.Errorf("linkio: read targets: %w", err)
	}

	return targets, nil
}

// ReadLinks reads JSON Lines: one JSON object per link. A bad record returns an
// error that names it, for example "record 3: ...". An empty input gives no links.
func ReadLinks(r io.Reader) ([]link.Link, error) {
	var links []link.Link

	dec := json.NewDecoder(r)

	for i := 1; ; i++ {
		var l link.Link

		err := dec.Decode(&l)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("linkio: record %v: %w", i, err)
		}
		links = append(links, l)
	}

	return links, nil
}

// WriteLinks writes JSON Lines: one link per line.
func WriteLinks(w io.Writer, links []link.Link) error {
	enc := json.NewEncoder(w)

	for i, l := range links {
		if err := enc.Encode(l); err != nil {
			return fmt.Errorf("linkio: record %v: %w", i+1, err)
		}
	}

	return nil
}

// WriteCSV writes the header id,code,target,owner_id,created_at,expires_at and one row per link.
// Times are RFC 3339 in UTC; a link that never expires has an empty expires_at.
func WriteCSV(w io.Writer, links []link.Link) error {
	cw := csv.NewWriter(w)
	
	var csvHeader = []string{"id", "code", "target", "owner_id", "created_at", "expires_at"}
	if err := cw.Write(csvHeader); err != nil {
		return fmt.Errorf("linkio: write csv header: %w", err)
	}

	for i, l := range links {
		row := []string {
			strconv.FormatUint(l.ID, 10),
			l.Code,
			l.Target,
			l.OwnerID,
			formatTime(l.CreatedAt),
			formatTime(l.ExpiresAt),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("linkio: write csv row %v: %w", i+1, err)
		}
	}
	
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("linkio: write csv: %w", err)
	}

	return nil
}

// LoadFile reads a JSON Lines file. A missing file returns an error that wraps fs.ErrNotExist.
func LoadFile(path string) ([]link.Link, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("linkio: load file %v: %w", path, fs.ErrNotExist)
	}
	if err != nil {
		return nil, fmt.Errorf("linkio: load file %v: %w", path, err)
	}
	defer f.Close()

	links, err := ReadLinks(f)
	if err != nil {
		return nil, fmt.Errorf("linkio: read file %v: %w", path, err)
	}

	return links, nil
}

// SaveFile creates (or replaces) the file at path and writes the links to it.
// The error from Close is part of the result.
func SaveFile(path string, links []link.Link) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("linkio: save file %v: %w", path, err)
	}

	bw := bufio.NewWriter(f)
	err = WriteLinks(bw, links)
	if err == nil {
		err = bw.Flush()
	}

	if err = errors.Join(err, f.Close()); err != nil {
		return fmt.Errorf("linkio: save file %v: %w", path, err)
	}

	return nil
}

// formatTime renders t as RFC 3339 in UTC, or "" for the zero time.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}
