package api

import (
	"backend/cryptopasta"
	"backend/storage"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	pseudoRand "math/rand"
	"net/http"
	"path/filepath"
	"time"

	"backend/types"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type TokenClaims struct {
	Id           string
	CurrentScore int
	Correct      string // encrypted correct answer
	Difficulty   types.Difficulty
	Jti          string
	Iat          int64
}

type QuizAPI struct {
	burnedIds          *bloom.BloomFilter
	burnedHighscoreIds *bloom.BloomFilter
	burnedJtis         *bloom.BloomFilter

	signatureKey  []byte
	encryptionKey [32]byte

	dataStore storage.Store
	manifests map[types.Difficulty]types.RandomManifest

	clipDir string

	mux *mux.Router
}

func NewQuizApi(manifests map[types.Difficulty]types.RandomManifest, dbPath, clipDir string) *QuizAPI {
	api := QuizAPI{}

	api.burnedIds = bloom.NewWithEstimates(10_000_000, 0.000001)
	api.burnedHighscoreIds = bloom.NewWithEstimates(10_000_000, 0.000001)
	api.burnedJtis = bloom.NewWithEstimates(10_000_000, 0.000001)

	nBig, err := rand.Int(rand.Reader, big.NewInt(27))

	if err != nil {
		panic(err)
	}
	pseudoRand.Seed(nBig.Int64())

	temp := make([]byte, 32)

	api.signatureKey = make([]byte, types.SIGNATURE_LENGTH)

	rand.Read(api.signatureKey)
	rand.Read(temp)
	copy(api.encryptionKey[:], temp)

	log.Printf("Signature Key: %s", base64.RawStdEncoding.EncodeToString(api.signatureKey))
	log.Printf("Encryption Key: %s", base64.RawStdEncoding.EncodeToString(api.encryptionKey[:]))

	api.clipDir = clipDir

	api.manifests = manifests
	api.dataStore.Init(dbPath)

	api.mux = mux.NewRouter()

	api.mux.HandleFunc("/clipquiz/v1/clip", func(w http.ResponseWriter, req *http.Request) {
		api.GetClipEndpoint(w, req)
	}).Methods(http.MethodPost)
	api.mux.HandleFunc("/clipquiz/v1/highscore", func(w http.ResponseWriter, req *http.Request) {
		api.RegisterHighscoreEndpoint(w, req)
	}).Methods(http.MethodPost)
	api.mux.HandleFunc("/clipquiz/v1/highscore", func(w http.ResponseWriter, req *http.Request) {
		api.GetHighScoresEndpoint(w, req)
	}).Methods(http.MethodGet)

	return &api
}

func (q *QuizAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	q.mux.ServeHTTP(w, req)
}

func (q *QuizAPI) parseFromJwt(authToken string) (TokenClaims, error) {
	token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("wrong algorithm: %v", token.Header["alg"])
		}

		return q.signatureKey, nil
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

		parsed.Difficulty = types.Difficulty(diffString)

		// parsed.Correct is a base64 encoded encrypted UTF-8 string
		encBytes, err := base64.StdEncoding.DecodeString(parsed.Correct)
		if err != nil {
			return TokenClaims{}, fmt.Errorf("failed to decode correct: base64 decoding error %s", err)
		}

		previousCorrectBytes, err := cryptopasta.Decrypt(encBytes, &q.encryptionKey)
		if err != nil {
			return TokenClaims{}, fmt.Errorf("failed to decrypt correct: %s", err)
		}

		parsed.Correct = string(previousCorrectBytes)

		var iatFloat float64
		if iatFloat, ok = claims["iat"].(float64); !ok {
			return TokenClaims{}, fmt.Errorf("iat not a float?")
		}

		parsed.Iat = int64(iatFloat)

		iat := time.Unix(int64(parsed.Iat), 0)

		if time.Since(iat) > time.Minute*15 {
			return TokenClaims{}, fmt.Errorf("token expired")
		}
		return parsed, nil
	}

	return TokenClaims{}, fmt.Errorf("claims parsing error")
}

func (q *QuizAPI) mintToken(claims TokenClaims) (string, error) {
	// encrypt correct
	var err error
	encryptedCorrect, err := cryptopasta.Encrypt([]byte(claims.Correct), &q.encryptionKey)

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
		"iat":          time.Now().Unix(),
	})

	tokenStr, err := token.SignedString(q.signatureKey)

	if err != nil {
		return "", fmt.Errorf("failed to sign token: %s", err)
	}

	return tokenStr, nil
}

func (q *QuizAPI) randomClip(diff types.Difficulty) (string, types.Episode) {
	manifest := q.manifests[diff]
	randomIndex := pseudoRand.Intn(len(manifest.Keys))
	clipName := manifest.Keys[randomIndex]
	correctEpisode := manifest.Lookup[clipName]

	return clipName, correctEpisode
}

func (q *QuizAPI) GetClipEndpoint(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Auth-Token")

	var claims TokenClaims
	var err error

	if auth == "" {
		// they're just starting out
		claims.Id = uuid.New().String()

		diff := req.URL.Query().Get("difficulty")

		if diff == "" || (diff != string(types.Easy) && diff != string(types.Medium) && diff != string(types.Hard) && diff != string(types.Legend)) {
			log.Printf("bad params on new request")
			http.Error(w, "failed to parse url", http.StatusBadRequest)
			return
		}

		claims.Difficulty = types.Difficulty(diff)

	} else {
		claims, err = q.parseFromJwt(auth)

		if err != nil {
			log.Printf("Token Parse Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if q.burnedIds.TestString(claims.Id) {
			log.Printf("id has been burned")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if q.burnedJtis.TestString(claims.Jti) {
			log.Printf("jti has been burned")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		q.burnedJtis.AddString(claims.Jti)

		// parse out their guess
		guess := req.URL.Query().Get("guess")

		if guess == "" {
			log.Printf("no guess")
			http.Error(w, "No guess", http.StatusBadRequest)
			return
		}

		if guess != claims.Correct {
			// that's all folks!
			// burn their id
			q.burnedIds.AddString(claims.Id)
			// remind them of their auth token
			w.Header().Add("Auth-Token", auth)
			// use 404 to indicate that they're done
			w.WriteHeader(http.StatusNotFound)

			// write the correct answer
			w.Write([]byte(claims.Correct))

			return
		}
		claims.CurrentScore += 1
	}

	// send a new file
	fileName, episode := q.randomClip(claims.Difficulty)
	claims.Correct = string(episode)

	// mint a new token
	auth, err = q.mintToken(claims)
	if err != nil {
		log.Printf("token creation error: %s", err)
		http.Error(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Auth-Token", auth)

	// serve the file
	filePath := filepath.Join(q.clipDir, fileName) + ".enc"
	log.Printf("serving %s", filePath)
	http.ServeFile(w, req, filePath)
}

func (q *QuizAPI) RegisterHighscoreEndpoint(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Auth-Token")

	if auth == "" {
		http.Error(w, "you need a token", http.StatusUnauthorized)
	}

	claims, err := q.parseFromJwt(auth)

	if err != nil {
		log.Printf("failed to validate token: %v", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	if q.burnedHighscoreIds.TestString(claims.Id) {
		log.Printf("attempted to register with burned token")
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	q.burnedHighscoreIds.AddString(claims.Id)

	name := req.URL.Query().Get("name")

	if name == "" || len(name) > 20 {
		log.Printf("invalid name attempted")
		http.Error(w, "that's not a valid name", http.StatusBadRequest)
		return
	}
	log.Printf("SCORE FROM CLAIM: %d", claims.CurrentScore)
	err = q.dataStore.RegisterScore(claims.Id, name, claims.Difficulty, claims.CurrentScore)

	if err != nil {
		log.Printf("failed to register score: %s", err)
		http.Error(w, "failed to register score", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Expires", time.Now().Add(time.Minute).Format(http.TimeFormat))
	w.WriteHeader(http.StatusCreated)
}

func (q *QuizAPI) GetHighScoresEndpoint(w http.ResponseWriter, req *http.Request) {
	highScores, err := q.dataStore.GetHighScores()
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

	w.Write(bytes)
}
