package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type HighScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type DifficultyHighScores struct {
	AllTime []HighScore `json:"allTime"`
	Today   []HighScore `json:"today"`
	Week    []HighScore `json:"week"`
}

type HighScores struct {
	HighScores map[string]DifficultyHighScores `json:"highScores"`
}

type Store struct {
	DatabaseFile string
	DB           *sql.DB
	Lock         sync.RWMutex

	LastQueried    *time.Time
	LastHighscores HighScores
}

func (s *Store) Init(file string) {
	s.DatabaseFile = file
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// make the db
		s.CreateDatabase()
	}

	db, err := sql.Open("sqlite3", file)

	if err != nil {
		log.Fatalf("failed to open db: %s", err)
	}
	s.DB = db
}

func (s *Store) CreateDatabase() {
	log.Printf("Database file `%q` not found, creating...", s.DatabaseFile)
	db, err := sql.Open("sqlite3", s.DatabaseFile)
	if err != nil {
		log.Fatalf("failed to open connection: %s", err)
	}
	defer db.Close()

	statements := []string{
		`CREATE TABLE "highscores" (
			"Id"	TEXT NOT NULL UNIQUE,
			"Score"	INTEGER NOT NULL,
			"Created"	TEXT NOT NULL,
			"Name"	TEXT NOT NULL,
			"Difficulty"	TEXT NOT NULL,
			PRIMARY KEY("Id")
		);`,
		`CREATE INDEX "DifficultyIndex" ON "highscores" (
			"Difficulty"
		);`,
		`CREATE INDEX "created" ON "highscores" (
			"Created"	DESC
		);`,
		`CREATE INDEX "mainindex" ON "highscores" (
			"Difficulty",
			"Created"	DESC,
			"Score"	DESC
		);`,
		`CREATE INDEX "otherindex" ON "highscores" (
			"Score"	DESC
		);`,
	}

	for i, stmt := range statements {
		_, err = db.Exec(stmt)

		if err != nil {
			log.Fatalf("could not execute create statement %d: %s", i, err)
		}
	}

}

func (s *Store) RegisterScore(id, name string, difficulty Difficulty, score int) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	_, err := s.DB.Exec(`
	INSERT INTO 
	 	highscores(Id, Score, Created, Name, Difficulty) 
	VALUES (?, ?, DATETIME('now', 'localtime'), ?, ?);`, id, score, name, string(difficulty))

	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	// reset cache
	s.LastQueried = nil

	return nil
}

func parseHighscores(query *sql.Rows) ([]HighScore, error) {
	rows := make([]HighScore, 0)

	for query.Next() {
		hs := HighScore{}

		err := query.Scan(&hs.Name, &hs.Score)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rows = append(rows, hs)
	}

	return rows, nil
}

func (s *Store) QueryForHighscores() (HighScores, error) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	allScores := HighScores{}
	allScores.HighScores = make(map[string]DifficultyHighScores, 4)

	for _, difficulty := range []Difficulty{Easy, Medium, Hard, Legend} {
		diffScores := DifficultyHighScores{}

		{
			rows, err := s.DB.Query(`
			SELECT
				Name, 
				Score 
			FROM highscores 
			WHERE difficulty = ? 
			ORDER BY Score DESC, Created ASC 
			LIMIT 10;
			`, string(difficulty))

			if err != nil {
				return HighScores{}, fmt.Errorf("failed to get alltime for difficulty %s: %w", string(difficulty), err)
			}

			scores, err := parseHighscores(rows)

			if err != nil {
				return HighScores{}, fmt.Errorf("parsing %s: %w", string(difficulty), err)
			}

			diffScores.AllTime = scores
		}

		{
			rows, err := s.DB.Query(`
				SELECT 
					Name, Score 
				FROM highscores 
				WHERE 
					Difficulty = ? AND 
					Created >= DATE('now', 'weekday 0', '-7 days', 'localtime') 
				ORDER BY Score DESC, Created ASC
				LIMIT 10;
			`, string(difficulty))

			if err != nil {
				return HighScores{}, fmt.Errorf("failed to get week for difficulty %s: %w", string(difficulty), err)
			}

			scores, err := parseHighscores(rows)

			if err != nil {
				return HighScores{}, fmt.Errorf("parsing %s: %w", string(difficulty), err)
			}

			diffScores.Week = scores
		}

		{
			rows, err := s.DB.Query(`
				SELECT 
					Name, Score 
				FROM highscores 
				WHERE 
					difficulty = ? AND 
					Created >= DATE('now', 'localtime') 
				ORDER BY Score DESC, Created ASC
				LIMIT 10;
			`, string(difficulty))

			if err != nil {
				return HighScores{}, fmt.Errorf("failed to get day for difficulty %s: %w", string(difficulty), err)
			}

			scores, err := parseHighscores(rows)

			if err != nil {
				return HighScores{}, fmt.Errorf("parsing %s: %w", string(difficulty), err)
			}

			diffScores.Today = scores
		}

		allScores.HighScores[string(difficulty)] = diffScores
	}

	return allScores, nil
}

func (s *Store) GetHighScores() (HighScores, error) {
	if s.LastQueried == nil || time.Since(*s.LastQueried) > time.Minute {
		allScores, err := s.QueryForHighscores()
		if err == nil {
			s.LastHighscores = allScores
			t := time.Now()
			s.LastQueried = &t
		}
		return allScores, err
	}

	// use cache
	return s.LastHighscores, nil
}
