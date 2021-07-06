package trie

import "testing"

func TestTrie(t *testing.T) {
	trie := TrieNode{}

	trie.Insert("test")
	if !trie.Contains("test") {
		t.Error("no contain :(")
	}
	if trie.Contains("tes") {
		t.Error("contain :(")
	}
	if trie.Contains("te") {
		t.Error("contain :(")
	}
	trie.Insert("tes")
	if !trie.Contains("tes") {
		t.Error("no contain :(")
	}
}
