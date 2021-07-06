package trie

import (
	"strings"
	"sync"
)

const start = '-'
const end = 'Z'

const SIZE = int(end) - int(start)

// SafeTrie locks the trie in a mutex so it's thread safe
type SafeTrie struct {
	root TrieNode
	lock sync.RWMutex
}

func (s *SafeTrie) Insert(str string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.root.Insert(str)
}

func (s *SafeTrie) Contains(str string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.root.Contains(str)
}

type TrieNode struct {
	children [SIZE]*TrieNode
	isEnd    bool
}

func (t *TrieNode) normalize(str string) []byte {
	str = strings.ToUpper(str)
	bytes := []byte(str)

	for i := range bytes {
		bytes[i] -= uint8(start)
	}
	return bytes
}

func (t *TrieNode) Insert(str string) {
	t.insert(t.normalize(str))
}

func (t *TrieNode) insert(value []uint8) {
	wordLength := len(value)
	current := t
	for i := 0; i < wordLength; i++ {
		index := value[i]
		if current.children[index] == nil {
			current.children[index] = &TrieNode{}
		}
		current = current.children[index]
	}
	current.isEnd = true
}

func (t *TrieNode) Contains(str string) bool {
	return t.contains(t.normalize(str))
}

func (t *TrieNode) contains(value []uint8) bool {
	wordLength := len(value)
	current := t
	for i := 0; i < wordLength; i++ {
		index := value[i]
		if current.children[index] == nil {
			return false
		}
		current = current.children[index]
	}
	if current.isEnd {
		return true
	}
	return false
}

func NewTrie() TrieNode {
	return TrieNode{}
}
