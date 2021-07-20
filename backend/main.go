package main

import (
	"backend/api"
	"backend/types"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/handlers"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

func LoadManifests(manifestPath string) map[types.Difficulty]types.RandomManifest {
	log.Printf("Manifest Dir: %s", manifestPath)

	manifests := make(map[types.Difficulty]types.RandomManifest, 6)

	for _, difficulty := range []types.Difficulty{types.Easy, types.Medium, types.Hard, types.Legend} {
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

		var manifest types.Manifest
		err = json.Unmarshal(bytes, &manifest)

		if err != nil {
			log.Panicf("failed to parse '%s': %s", fullPath, err)
		}

		var rm types.RandomManifest
		totalSize := manifest.TotalSize()

		rm.Lookup = make(map[string]types.Episode, totalSize)
		rm.Keys = make([]string, totalSize)

		total := 0
		for i := range manifest.PhantomMenace {
			rm.Lookup[manifest.PhantomMenace[i]] = types.PhantomMenace
			rm.Keys[total] = manifest.PhantomMenace[i]
			total++
		}

		for i := range manifest.AttackClones {
			rm.Lookup[manifest.AttackClones[i]] = types.AttackClones
			rm.Keys[total] = manifest.AttackClones[i]
			total++
		}

		for i := range manifest.RevengeSith {
			rm.Lookup[manifest.RevengeSith[i]] = types.RevengeSith
			rm.Keys[total] = manifest.RevengeSith[i]
			total++
		}

		for i := range manifest.NewHope {
			rm.Lookup[manifest.NewHope[i]] = types.NewHope
			rm.Keys[total] = manifest.NewHope[i]
			total++
		}

		for i := range manifest.Empire {
			rm.Lookup[manifest.Empire[i]] = types.Empire
			rm.Keys[total] = manifest.Empire[i]
			total++
		}

		for i := range manifest.Rotj {
			rm.Lookup[manifest.Rotj[i]] = types.Rotj
			rm.Keys[total] = manifest.Rotj[i]
			total++
		}

		if total != totalSize || len(rm.Keys) != len(rm.Lookup) {
			log.Panic("AAAHHHH")
		}

		manifests[difficulty] = rm
	}

	return manifests
}

const PREFIX = "/clipquiz/v1/"

func main() {
	log.Print("Hello There")

	// READ CONFIGURATION
	manifestPath := os.Getenv("MANIFEST_FILE_LOCATION")
	if manifestPath == "" {
		manifestPath = "."
	}

	clipDir := os.Getenv("CLIP_DIRECTORY")

	dbPath := os.Getenv("DATABASE_FILE")
	if dbPath == "" {
		dbPath = "highscores.db"
	}

	frontendOrigin := os.Getenv("BACKEND_FRONTEND_ALLOWED_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:8000"
	}

	fmt.Printf("Configuration:\n\tManifest Path = '%s'\n\tClip Dir = '%s'\n\tDB Path = '%s'\n\tFrontend Origin = '%s'\n", manifestPath, clipDir, dbPath, frontendOrigin)

	// get the manifest files
	manifests := LoadManifests(manifestPath)

	quizApi := api.NewQuizApi(manifests, dbPath, clipDir)

	var debug = false
	if os.Getenv("DEBUG") != "" {
		debug = true
	}

	// init middleware
	cors := cors.New(cors.Options{
		AllowedHeaders: []string{"Auth-Token"},
		ExposedHeaders: []string{"Auth-Token"},
		AllowedOrigins: []string{frontendOrigin, "http://192.168.1.29:8000"},
		Debug:          debug,
	})
	handler := cors.Handler(quizApi)
	handler = handlers.CombinedLoggingHandler(os.Stdout, handler)

	log.Print("Listening on port 3123...")

	s := &http.Server{
		Addr:           ":3123",
		Handler:        handler,
		MaxHeaderBytes: 2048,
	}

	s.ListenAndServe()
}
