package main

import (
	"backend/cryptopasta"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	pseudoRand "math/rand"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type TokenClaims struct {
	Id           string
	CurrentScore int
	Correct      string // encrypted correct answer
	Difficulty   Difficulty
	Jti          string
}

func parseFromJwt(authToken string) (TokenClaims, error) {
	token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("wrong algorithm: %v", token.Header["alg"])
		}

		return signatureKey, nil
	})

	if err != nil {
		return TokenClaims{}, fmt.Errorf("parse error: %s", err)
	}

	var parsed TokenClaims

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

		// type checking generic interfaces!
		if parsed.Id, ok = claims["id"].(string); !ok {
			return TokenClaims{}, fmt.Errorf("id not a string?")
		}

		if parsed.Jti, ok = claims["jti"].(string); !ok {
			return TokenClaims{}, fmt.Errorf("jti not a string?")
		}

		var scoreFloat float64
		if scoreFloat, ok = claims["currentScore"].(float64); !ok {
			return TokenClaims{}, fmt.Errorf("currentScore not a float?")
		}

		parsed.CurrentScore = int(scoreFloat)

		if parsed.Correct, ok = claims["correct"].(string); !ok {
			return TokenClaims{}, fmt.Errorf("correct not a string?")
		}

		var diffString string
		if diffString, ok = claims["difficulty"].(string); !ok {
			return TokenClaims{}, fmt.Errorf("difficulty not a string?")
		}

		parsed.Difficulty = Difficulty(diffString)

		// parsed.Correct is a base64 encoded encrypted UTF-8 string
		encBytes, err := base64.StdEncoding.DecodeString(parsed.Correct)
		if err != nil {
			return TokenClaims{}, fmt.Errorf("failed to decode correct: base64 decoding error %s", err)
		}

		previousCorrectBytes, err := cryptopasta.Decrypt(encBytes, &encryptionKey)
		if err != nil {
			return TokenClaims{}, fmt.Errorf("failed to decrypt correct: %s", err)
		}

		parsed.Correct = string(previousCorrectBytes)

		return parsed, nil
	}

	return TokenClaims{}, fmt.Errorf("claims parsing error")
}

func mintToken(claims TokenClaims) (string, error) {
	// encrypt correct
	var err error
	encryptedCorrect, err := cryptopasta.Encrypt([]byte(claims.Correct), &encryptionKey)

	if err != nil {
		return "", fmt.Errorf("failed to encrypt correct: %s", err)
	}

	claims.Correct = base64.StdEncoding.EncodeToString(encryptedCorrect)

	jti := uuid.New().String()

	// make a new token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":           claims.Id,
		"correct":      claims.Correct,
		"currentScore": claims.CurrentScore,
		"difficulty":   claims.Difficulty,
		"jti":          jti,
	})

	tokenStr, err := token.SignedString(signatureKey)

	if err != nil {
		return "", fmt.Errorf("failed to sign token: %s", err)
	}

	return tokenStr, nil
}

func randomClip(diff Difficulty) (string, Episode) {
	manifest := Manifests[diff]
	randomIndex := pseudoRand.Intn(len(manifest.keys))
	clipName := manifest.keys[randomIndex]
	correctEpisode := manifest.lookup[clipName]

	return clipName, correctEpisode
}

func getClip(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Authorization")

	var claims TokenClaims
	var err error

	if auth == "" {
		// they're just starting out
		claims.Id = uuid.New().String()

		diff := req.URL.Query().Get("difficulty")

		if diff == "" || (diff != string(Easy) && diff != string(Medium) && diff != string(Hard) && diff != string(Legend)) {
			log.Printf("bad params on new request")
			http.Error(w, "failed to parse url", http.StatusBadRequest)
			return
		}

		claims.Difficulty = Difficulty(diff)

	} else {
		tokenSplit := strings.Split(auth, "Bearer ")

		if len(tokenSplit) != 2 {
			log.Printf("invalid bearer token: no data? %v", tokenSplit)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		auth = tokenSplit[1]
		claims, err = parseFromJwt(auth)

		if err != nil {
			log.Printf("Token Parse Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if burnedIds.Contains(claims.Id) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if burnedJtis.Contains(claims.Jti) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		burnedJtis.Insert(claims.Jti)

		// parse out their guess
		guess := req.URL.Query().Get("guess")

		if guess == "" {
			http.Error(w, "No guess", http.StatusBadRequest)
			return
		}

		if guess != claims.Correct {
			// that's all folks!
			// burn their id
			burnedIds.Insert(claims.Id)
			// remind them of their auth token
			w.Header().Add("Authorization", fmt.Sprintf("Bearer %s", auth))
			// use 404 to indicate that they're done
			w.WriteHeader(http.StatusNotFound)

			// write the correct answer
			w.Write([]byte(claims.Correct))

			return
		}
		claims.CurrentScore += 1
	}

	// send a new file
	fileName, episode := randomClip(claims.Difficulty)
	episode = NewHope
	claims.Correct = string(episode)

	// mint a new token
	auth, err = mintToken(claims)
	if err != nil {
		log.Printf("token creation error: %s", err)
	}

	w.Header().Add("Authorization", fmt.Sprintf("Bearer %s", auth))

	// serve the file
	filePath := filepath.Join(ClipDir, fileName) + ".enc"
	log.Printf("serving %s", filePath)
	http.ServeFile(w, req, filePath)
}

func registerHighscore(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Authorization")

	if auth == "" {
		http.Error(w, "you need a token", http.StatusUnauthorized)
	}

	tokenSplit := strings.Split(auth, "Bearer ")

	if len(tokenSplit) != 2 {
		log.Printf("invalid bearer token: no data? %v", tokenSplit)
		http.Error(w, "invalid auth token", http.StatusUnauthorized)
		return
	}

	auth = tokenSplit[1]

	claims, err := parseFromJwt(auth)

	if err != nil {
		log.Printf("failed to validate token: %v", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	if burnedHighscoreIds.Contains(claims.Id) {
		log.Printf("attempted to register with burned token")
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	burnedHighscoreIds.Insert(claims.Id)

	name := req.URL.Query().Get("name")

	if name == "" || len(name) > 20 {
		log.Printf("invalid name attempted")
		http.Error(w, "that's not a valid name", http.StatusBadRequest)
		return
	}

	err = DataStore.RegisterScore(claims.Id, name, claims.Difficulty, claims.CurrentScore)

	if err != nil {
		log.Printf("failed to register score: %s", err)
		http.Error(w, "Failed to register score", http.StatusInternalServerError)
		return
	}
}

func GetHighScores(w http.ResponseWriter, req *http.Request) {
	highScores, err := DataStore.GetHighScores()
	if err != nil {
		log.Printf("failed to get high scores: %s", err)
		http.Error(w, "failed to get high scores!", http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(&highScores)
	if err != nil {
		log.Printf("failed to marshall high scores: %s", err)
		http.Error(w, "failed to marshall high scores!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// todo: caching!
	w.Write(bytes)
}
