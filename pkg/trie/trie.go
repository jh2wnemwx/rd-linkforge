// Package trie is a prefix tree over the bytes of its words, used to
// autocomplete aliases.
package trie

// Trie holds a set of non-empty words. The zero value is not usable; call New.
type Trie struct {
	root *node
}

// node is one byte position. Children are kept in a 256-slot array so that a
// walk in index order visits words in ascending byte order without sorting.
type node struct {
	children [256]*node
	count    int  // number of children that are not nil
	word     bool // a word ends here
}

// New returns an empty trie.
func New() *Trie {
	return &Trie{root: &node{}}
}

// Insert adds word. It returns false, and changes nothing, when word is empty
// or already there.
func (t *Trie) Insert(word string) bool {
	if word == "" {
		return false
	}
	n := t.root
	for i := 0; i < len(word); i++ {
		b := word[i]
		if n.children[b] == nil {
			n.children[b] = &node{}
			n.count++
		}
		n = n.children[b]
	}
	if n.word {
		return false
	}
	n.word = true
	return true
}

// Contains reports whether word is in the trie.
func (t *Trie) Contains(word string) bool {
	if word == "" {
		return false
	}
	n := t.find(word)
	return n != nil && n.word
}

// Delete removes word and reports whether it was there. Nodes that no longer
// lead to any word are pruned; a longer word that starts with word is kept.
func (t *Trie) Delete(word string) bool {
	if word == "" {
		return false
	}

	// Record the path so that empty nodes can be pruned bottom-up.
	path := make([]*node, 0, len(word)+1)
	n := t.root
	path = append(path, n)
	for i := 0; i < len(word); i++ {
		n = n.children[word[i]]
		if n == nil {
			return false
		}
		path = append(path, n)
	}
	if !n.word {
		return false
	}
	n.word = false

	for i := len(word); i > 0; i-- {
		child := path[i]
		if child.word || child.count > 0 {
			break
		}
		parent := path[i-1]
		parent.children[word[i-1]] = nil
		parent.count--
	}
	return true
}

// WithPrefix returns the words that start with prefix (the prefix itself too, if it
// is a word), in ascending byte order. A limit of zero or less means no limit.
func (t *Trie) WithPrefix(prefix string, limit int) []string {
	words := []string{}
	n := t.find(prefix)
	if n == nil {
		return words
	}

	buf := []byte(prefix)
	var walk func(n *node) bool // returns false once the limit is reached
	walk = func(n *node) bool {
		if n.word {
			words = append(words, string(buf))
			if limit > 0 && len(words) >= limit {
				return false
			}
		}
		for b, child := range n.children {
			if child == nil {
				continue
			}
			buf = append(buf, byte(b))
			more := walk(child)
			buf = buf[:len(buf)-1]
			if !more {
				return false
			}
		}
		return true
	}
	walk(n)
	return words
}

// find returns the node reached by s, or nil. The empty string is the root.
func (t *Trie) find(s string) *node {
	n := t.root
	for i := 0; i < len(s) && n != nil; i++ {
		n = n.children[s[i]]
	}
	return n
}
