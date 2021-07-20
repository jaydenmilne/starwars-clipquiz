package types

type RandomManifest struct {
	Lookup map[string]Episode
	Keys   []string
}

func (m *Manifest) TotalSize() int {
	return len(m.PhantomMenace) + len(m.AttackClones) + len(m.RevengeSith) + len(m.NewHope) + len(m.Empire) + len(m.Rotj)
}

var ClipDir string

// ////////////////////////////////////////
type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
	Legend Difficulty = "legend"
)

type Episode string

var (
	PhantomMenace Episode = "phantom-menace"
	AttackClones  Episode = "attack-clones"
	RevengeSith   Episode = "revenge-sith"
	NewHope       Episode = "new-hope"
	Empire        Episode = "empire"
	Rotj          Episode = "rotj"
)

type Manifest struct {
	PhantomMenace []string `json:"phantom-menace"`
	AttackClones  []string `json:"attack-clones"`
	RevengeSith   []string `json:"revenge-sith"`
	NewHope       []string `json:"new-hope"`
	Empire        []string `json:"empire"`
	Rotj          []string `json:"rotj"`
}

const SIGNATURE_LENGTH = 32
