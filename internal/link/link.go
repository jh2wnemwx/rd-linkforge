package link

import (
	"errors"
	"net/url"
	"time"
	"github.com/jh2wnemwx/rd-linkforge/internal/base62"
)

var (
	ErrEmptyTarget			= errors.New("link: empty target")
	ErrUnsupportedScheme	= errors.New("link: unsupported scheme")
)

type Link struct {
    ID        uint64
    Code      string
    Target    string
    CreatedAt time.Time
}

// New builds a link, validating the target address.
func New(id uint64, target string) (Link, error) {
	if target == "" {
		return Link{}, ErrEmptyTarget
	}

	if !isValidURL(target) {
		return Link{}, ErrUnsupportedScheme
	}

	return Link {
		ID: id,
		Code: base62.EncodeWidth(id, 6),
		Target: target,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}