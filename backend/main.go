package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	pseudoRand "math/rand"
	"net/http"
	"os"
	"path"
	"time"

	"backend/trie"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

const SIGNATURE_LENGTH = 32

// set-once globals
var signatureKey []byte
var encryptionKey [32]byte

var burnedIds trie.SafeTrie
var burnedHighscoreIds trie.SafeTrie
var burnedJtis trie.SafeTrie

type RandomManifest struct {
	lookup map[string]Episode
	keys   []string
}

var Manifests map[Difficulty]RandomManifest

var ClipDir string

var DataStore Store

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

func (m *Manifest) TotalSize() int {
	return len(m.PhantomMenace) + len(m.AttackClones) + len(m.RevengeSith) + len(m.NewHope) + len(m.Empire) + len(m.Rotj)
}

func LoadManifests() {
	manifestPath := os.Getenv("MANIFEST_FILE_LOCATION")
	if manifestPath == "" {
		manifestPath = "."
	}

	log.Printf("Manifest Dir: %s", manifestPath)

	Manifests = make(map[Difficulty]RandomManifest, 6)

	for _, difficulty := range []Difficulty{Easy, Medium, Hard, Legend} {
		fullPath := path.Join(manifestPath, fmt.Sprintf("%s.json", string(difficulty)))
		jsonFile, err := os.Open(fullPath)
		if err != nil {
			log.Panicf("failed to open '%s': %s", fullPath, err)
		}
		defer jsonFile.Close()

		bytes, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			log.Panicf("failed to read '%s': %s", fullPath, err)
		}

		var manifest Manifest
		err = json.Unmarshal(bytes, &manifest)

		if err != nil {
			log.Panicf("failed to parse '%s': %s", fullPath, err)
		}

		var rm RandomManifest
		totalSize := manifest.TotalSize()

		rm.lookup = make(map[string]Episode, totalSize)
		rm.keys = make([]string, totalSize)

		total := 0
		for i := range manifest.PhantomMenace {
			rm.lookup[manifest.PhantomMenace[i]] = PhantomMenace
			rm.keys[total] = manifest.PhantomMenace[i]
			total++
		}

		for i := range manifest.AttackClones {
			rm.lookup[manifest.AttackClones[i]] = AttackClones
			rm.keys[total] = manifest.AttackClones[i]
			total++
		}

		for i := range manifest.RevengeSith {
			rm.lookup[manifest.RevengeSith[i]] = RevengeSith
			rm.keys[total] = manifest.RevengeSith[i]
			total++
		}

		for i := range manifest.NewHope {
			rm.lookup[manifest.NewHope[i]] = NewHope
			rm.keys[total] = manifest.NewHope[i]
			total++
		}

		for i := range manifest.Empire {
			rm.lookup[manifest.Empire[i]] = Empire
			rm.keys[total] = manifest.Empire[i]
			total++
		}

		for i := range manifest.Rotj {
			rm.lookup[manifest.Rotj[i]] = Rotj
			rm.keys[total] = manifest.Rotj[i]
			total++
		}

		if total != totalSize || len(rm.keys) != len(rm.lookup) {
			log.Panic("AAAHHHH")
		}

		Manifests[difficulty] = rm
	}
}

const PREFIX = "/clipquiz/v1/"

func main() {
	log.Print("Hello There")
	pseudoRand.Seed(time.Now().UTC().UnixNano())
	signatureKey = make([]byte, SIGNATURE_LENGTH)

	temp := make([]byte, 32)

	rand.Read(signatureKey)
	rand.Read(temp)

	copy(encryptionKey[:], temp)

	// get the manifest files
	LoadManifests()

	// get the clip directory
	ClipDir = os.Getenv("CLIP_DIRECTORY")
	log.Printf("Clip Dir: %s", ClipDir)
	mux := mux.NewRouter()

	mux.HandleFunc("/clipquiz/v1/requestclip", getClip).Methods(http.MethodPost).Headers()
	mux.HandleFunc("/clipquiz/v1/registerHighscore", registerHighscore).Methods(http.MethodPost)
	mux.HandleFunc("/clipquiz/v1/highScores", GetHighScores).Methods(http.MethodGet)

	// init db
	dbPath := os.Getenv("DATABASE_FILE")
	if dbPath == "" {
		dbPath = "highscores.db"
	}

	log.Printf("DB Store: %s", dbPath)
	DataStore.Init(dbPath)

	frontendOrigin := os.Getenv("BACKEND_FRONTEND_ALLOWED_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:8000"
	}

	var debug = false
	if os.Getenv("DEBUG") != "" {
		debug = true
	}

	// init middleware
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{frontendOrigin},
		AllowCredentials: true,
		Debug:            debug,
	})
	handler := cors.Handler(mux)
	handler = handlers.CombinedLoggingHandler(os.Stdout, handler)

	log.Print("Listening on port 3000...")
	http.ListenAndServe(":3000", mux)

	s := http.Server{
		Addr:    ":3000",
		Handler: handler,
	}

	s.ListenAndServe()
}
