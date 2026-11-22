package trie

import (
	"slices"
	"testing"
)

func newTrie(t *testing.T, words ...string) *Trie {
	t.Helper()
	tr := New()
	for _, w := range words {
		if !tr.Insert(w) {
			t.Fatalf("Insert(%q) = false", w)
		}
	}
	return tr
}

var exampleWords = []string{"docs", "doc", "dog", "do-it", "day", "api", "dock"}

func TestInsertAndContains(t *testing.T) {
	tr := newTrie(t, exampleWords...)

	if tr.Insert("doc") {
		t.Error(`Insert("doc") again = true, want false`)
	}
	if tr.Insert("") {
		t.Error(`Insert("") = true, want false`)
	}

	for _, w := range exampleWords {
		if !tr.Contains(w) {
			t.Errorf("Contains(%q) = false", w)
		}
	}
	for _, w := range []string{"", "d", "do", "docx", "apis", "x"} {
		if tr.Contains(w) {
			t.Errorf("Contains(%q) = true", w)
		}
	}
}

func TestDelete(t *testing.T) {
	tr := newTrie(t, exampleWords...)

	if tr.Delete("") {
		t.Error(`Delete("") = true`)
	}
	if tr.Delete("do") || tr.Delete("docx") || tr.Delete("zzz") {
		t.Error("Delete of a missing word = true")
	}

	// doc is a prefix of dock and docs: deleting it keeps them.
	if !tr.Delete("doc") {
		t.Fatal(`Delete("doc") = false`)
	}
	if tr.Contains("doc") {
		t.Error("doc still there after Delete")
	}
	if !tr.Contains("dock") || !tr.Contains("docs") {
		t.Error("deleting doc removed a longer word")
	}
	if tr.Delete("doc") {
		t.Error(`second Delete("doc") = true`)
	}

	// Deleting a longer word keeps its prefix word.
	tr.Insert("doc")
	tr.Delete("docs")
	tr.Delete("dock")
	if !tr.Contains("doc") {
		t.Error("deleting dock and docs removed doc")
	}

	// A word can be inserted again after it was deleted.
	if !tr.Insert("docs") {
		t.Error(`Insert("docs") after Delete = false`)
	}
}

func TestDeletePrunesNodes(t *testing.T) {
	tr := newTrie(t, "abc")
	tr.Delete("abc")
	if tr.root.count != 0 {
		t.Errorf("root still has %d children after deleting the only word", tr.root.count)
	}
	if got := tr.WithPrefix("", 0); len(got) != 0 {
		t.Errorf("WithPrefix(\"\") = %v, want empty", got)
	}
}

func TestWithPrefix(t *testing.T) {
	tr := newTrie(t, exampleWords...)

	tests := []struct {
		prefix string
		limit  int
		want   []string
	}{
		{"do", 0, []string{"do-it", "doc", "dock", "docs", "dog"}},
		{"do", 2, []string{"do-it", "doc"}},
		{"do", -1, []string{"do-it", "doc", "dock", "docs", "dog"}},
		{"doc", 0, []string{"doc", "dock", "docs"}}, // the prefix itself is a word
		{"docs", 0, []string{"docs"}},
		{"x", 0, []string{}},
		{"docsx", 0, []string{}},
		{"", 0, []string{"api", "day", "do-it", "doc", "dock", "docs", "dog"}},
		{"", 1, []string{"api"}},
		{"do", 100, []string{"do-it", "doc", "dock", "docs", "dog"}},
	}
	for _, tt := range tests {
		got := tr.WithPrefix(tt.prefix, tt.limit)
		if got == nil || !slices.Equal(got, tt.want) {
			t.Errorf("WithPrefix(%q, %d) = %#v, want %#v", tt.prefix, tt.limit, got, tt.want)
		}
	}
}

func TestWithPrefixByteOrder(t *testing.T) {
	// Byte order: upper case before lower case, and multi-byte UTF-8 last.
	tr := newTrie(t, "b", "B", "é", "a", "A", "0")
	want := []string{"0", "A", "B", "a", "b", "é"}
	if got := tr.WithPrefix("", 0); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
