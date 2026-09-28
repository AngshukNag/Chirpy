package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/AngshukNag/Chirpy/internal/auth"
	"github.com/AngshukNag/Chirpy/internal/database"
	"github.com/AngshukNag/Chirpy/internal/utilities"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
	platform       string
	userId         uuid.UUID
}

type ChirpyError struct {
	Error string `json:"error"`
}

type ValidSuccess struct {
	Valid bool `json:"valid"`
}

type CreateUserRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUserResponseBody struct {
	Id           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

type AddChirpRequest struct {
	Body string `json:"body"`
}

type AddChirpResponse struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type RefreshTokenResponse struct {
	Token string `json:"token"`
}

type PolkaChirpyRedWebhookRequest struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (apiCtx *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/admin/reset" {
			apiCtx.fileserverHits.Store(0)
		} else {
			apiCtx.fileserverHits.Add(1)
		}
		next.ServeHTTP(w, req)
	})
}

func handleHealthzHTTPRequests(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func sendChirpError(w http.ResponseWriter, err string, statusCode int) {
	chirpError := ChirpyError{
		Error: err,
	}
	errorData, errDecode := json.Marshal(chirpError)
	if errDecode == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(errorData)
	} else {
		w.WriteHeader(statusCode)
	}
	return
}

func handleAddChirpHTTPRequest(w http.ResponseWriter, req *http.Request) {
	authToken, err := auth.GetToken(req.Header, auth.BearerToken)
	fmt.Println("Auth token received for add Chirp request: ", authToken)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")

	userID, err := auth.ValidateJWT(authToken, jwtSecret)

	if err != nil {
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(req.Body)
	addChirp := AddChirpRequest{}
	err_decode := decoder.Decode(&addChirp)

	if err_decode != nil {
		sendChirpError(w, fmt.Sprintf("Chirp request error: %v", err_decode), 500)
		return
	}

	if len(addChirp.Body) > 140 {
		sendChirpError(w, "Chirp is too long", 400)
	} else {
		chirps, err := apiContext.queries.AddChirp(req.Context(), database.AddChirpParams{
			Body:   addChirp.Body,
			UserID: userID,
		})

		if err != nil || len(chirps) == 0 {
			var err_string string
			if err != nil {
				err_string = fmt.Sprintf("Error adding chirp: %v", err)
			} else {
				err_string = "Chirp length cannot be zero"
			}
			sendChirpError(w, err_string, 400)
			return
		}

		chirp := chirps[0]

		chirpResponse := AddChirpResponse{
			Id:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

		responseData, err := json.Marshal(chirpResponse)

		if err != nil {
			sendChirpError(w, fmt.Sprintf("Internal Server Error:  %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(responseData)
	}
}

func handleGetAllChirpsHTTPRequest(w http.ResponseWriter, req *http.Request) {
	authorID, err_authorID_parse := uuid.Parse(req.URL.Query().Get("author_id"))
	sortType := req.URL.Query().Get("sort")

	var chirps []database.Chirp
	var chirpQueryError error

	sortDirection := "asc"
	if sortType == "desc" || sortType == "asc" {
		sortDirection = sortType
	}

	if authorID.String() == "" || err_authorID_parse != nil {
		chirps, chirpQueryError = apiContext.queries.GetAllChirps(req.Context(), sortDirection)
	} else {
		chirps, chirpQueryError = apiContext.queries.GetChirpsForAuthor(req.Context(), database.GetChirpsForAuthorParams{
			UserID:    authorID,
			SortOrder: sortDirection,
		})
	}

	if chirpQueryError != nil {
		sendChirpError(w, fmt.Sprintf("Error retreiving chirps: %v", chirpQueryError), 500)
		return
	}

	chirpsResponse := utilities.Map(chirps, func(chirp database.Chirp) AddChirpResponse {
		return AddChirpResponse{
			Id:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
	})

	responseData, err_response_marshall := json.Marshal(chirpsResponse)

	if err_response_marshall != nil {
		sendChirpError(w, "Internal Server Error", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(responseData)
}

func handleGetChirpHTTPRequest(w http.ResponseWriter, req *http.Request) {
	chirpId := req.PathValue("chirpID")
	chirpUUID, err := uuid.Parse(chirpId)
	if err != nil {
		sendChirpError(w, fmt.Sprintf("Chirp not found: %v", err), 404)
		return
	}

	chirp, err := apiContext.queries.GetChirp(req.Context(), chirpUUID)

	if err != nil {
		sendChirpError(w, fmt.Sprintf("Chirp not found: %v", err), 404)
		return
	}

	chirpResponse := AddChirpResponse{
		Id:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	responseData, err := json.Marshal(chirpResponse)

	if err != nil {
		sendChirpError(w, fmt.Sprintf("Internal Server Error: %v", err), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(responseData)
}

func removeProfaneWords(input string) string {
	profaneWords := map[string]struct{}{
		"kerfuffle": struct{}{},
		"sharbert":  struct{}{},
		"fornax":    struct{}{},
	}

	words := strings.Fields(input)

	for index, word := range words {
		if _, ok := profaneWords[strings.ToLower(word)]; ok == true {
			words[index] = "****"
		}
	}

	return strings.Join(words, " ")
}

func handleMetricsAndResetHTTPRequest(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/admin/metrics" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, apiContext.fileserverHits.Load())))
	} else if req.URL.Path == "/admin/reset" {
		if apiContext.platform != "dev" {
			w.WriteHeader(403)
			return
		}

		err := apiContext.queries.DeleteUsers(req.Context())
		if err != nil {
			sendChirpError(w, fmt.Sprintf("Error deleting users: %v", err), 400)
			return
		}

		w.WriteHeader(200)
	}
}

func handleCreateUserHTTPRequest(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	newUser := CreateUserRequestBody{}

	err := decoder.Decode(&newUser)

	if err != nil {
		sendChirpError(w, "Something went wrong", 500)
		return
	}

	hashed_password, err := auth.HashPassword(newUser.Password)

	if err != nil {
		sendChirpError(w, fmt.Sprintf("Internal Server Error: %v", err), 500)
		return
	}

	user, err := apiContext.queries.CreateUser(req.Context(), database.CreateUserParams{
		Email: newUser.Email,
		HashedPassword: sql.NullString{
			String: hashed_password,
			Valid:  true,
		},
	})

	if err != nil {
		sendChirpError(w, fmt.Sprintf("Error creating user: %v", err), 400)
		return
	}

	responseBody := CreateUserResponseBody{
		Id:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.CreatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}

	responseData, err := json.Marshal(responseBody)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(responseData)
}

func handleUpdateUserHTTPRequest(w http.ResponseWriter, req *http.Request) {
	accessToken, err := auth.GetToken(req.Header, auth.BearerToken)
	if err != nil {
		sendChirpError(w, "Invalid access token", 401)
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")

	userID, err_validate_token := auth.ValidateJWT(accessToken, jwtSecret)

	if err_validate_token != nil {
		sendChirpError(w, "Invalid access token", 401)
		return
	}

	decoder := json.NewDecoder(req.Body)
	newUser := CreateUserRequestBody{}

	err_decode := decoder.Decode(&newUser)

	if err_decode != nil {
		sendChirpError(w, "Something went wrong", 500)
		return
	}

	hashed_password, err := auth.HashPassword(newUser.Password)

	if err != nil {
		sendChirpError(w, fmt.Sprintf("Internal Server Error: %v", err), 500)
		return
	}

	user, err := apiContext.queries.UpdateUser(req.Context(), database.UpdateUserParams{
		ID:    userID,
		Email: newUser.Email,
		HashedPassword: sql.NullString{
			String: hashed_password,
			Valid:  true,
		},
	})

	if err != nil {
		sendChirpError(w, fmt.Sprintf("Error creating user: %v", err), 401)
		return
	}

	responseBody := CreateUserResponseBody{
		Id:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.CreatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}

	responseData, err := json.Marshal(responseBody)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(responseData)
}

func handleLoginHTTPRequest(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	newUser := CreateUserRequestBody{}

	err := decoder.Decode(&newUser)

	if err != nil {
		sendChirpError(w, "Something went wrong", 500)
		return
	}

	user, err := apiContext.queries.GetUser(req.Context(), newUser.Email)

	if err != nil || user.HashedPassword.Valid == false {
		sendChirpError(w, fmt.Sprintf("Incorrect email or password"), 401)
		return
	}

	hashedPassword := user.HashedPassword.String
	match, err := auth.CheckPasswordHash(newUser.Password, hashedPassword)

	if !match || err != nil {
		sendChirpError(w, fmt.Sprintf("Incorrect email or password"), 401)
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	token_duration := 3600
	// duration, _ := time.ParseDuration(fmt.Sprintf("%vs", token_duration))
	// token, err := auth.MakeJWT(user.ID, jwtSecret, duration)
	token, err := makeJWTAuthTokenFor(user.ID, jwtSecret, token_duration)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	refreshToken := auth.MakeRefreshToken()
	refreshTokenParams := database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
	}

	refreshTokenResult, err := apiContext.queries.CreateRefreshToken(req.Context(), refreshTokenParams)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	userResponse := CreateUserResponseBody{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.CreatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshTokenResult.Token,
		IsChirpyRed:  user.IsChirpyRed,
	}

	responseData, err := json.Marshal(userResponse)

	if err != nil {
		sendChirpError(w, fmt.Sprintf("Internal Server Error"), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(responseData)
}

func makeJWTAuthTokenFor(userID uuid.UUID, jwtSecret string, tokenDuration int) (string, error) {
	duration, _ := time.ParseDuration(fmt.Sprintf("%vs", tokenDuration))
	return auth.MakeJWT(userID, jwtSecret, duration)
}

func handleRefreshTokenHTTPRequest(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetToken(req.Header, auth.BearerToken)

	if err != nil {
		sendChirpError(w, "Invalid refresh token", 400)
		return
	}

	userID, err := apiContext.queries.GetUserFromRefreshToken(req.Context(), refreshToken)

	if err != nil {
		sendChirpError(w, "Invalid refresh token", 401)
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	token_duration := 3600
	token, err := makeJWTAuthTokenFor(userID, jwtSecret, token_duration)

	response := RefreshTokenResponse{
		Token: token,
	}

	responseData, err := json.Marshal(response)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(responseData)
}

func handleRefreshTokenRevokeHTTPRequest(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetToken(req.Header, auth.BearerToken)

	if err != nil {
		sendChirpError(w, "Invalid refresh token", 400)
		return
	}

	err_revoke := apiContext.queries.RevokeRefreshToken(req.Context(), refreshToken)

	if err_revoke != nil {
		sendChirpError(w, "Invalid refresh token", 401)
		return
	}

	w.WriteHeader(204)
}

func handleDeleteChirpHTTPRequest(w http.ResponseWriter, req *http.Request) {
	chirpId := req.PathValue("chirpID")
	chirpUUID, err := uuid.Parse(chirpId)
	if err != nil {
		sendChirpError(w, fmt.Sprintf("Chirp not found: %v", err), 404)
		return
	}

	accessToken, err := auth.GetToken(req.Header, auth.BearerToken)
	if err != nil {
		sendChirpError(w, "invalid access token", 401)
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	userID, err := auth.ValidateJWT(accessToken, jwtSecret)

	if err != nil {
		sendChirpError(w, "Unauthorized access", 403)
		return
	}

	chirp, err := apiContext.queries.GetChirp(req.Context(), chirpUUID)
	if err != nil {
		sendChirpError(w, "Chirp does not exist", 404)
		return
	}

	if chirp.UserID != userID {
		sendChirpError(w, "Unauthorized access", 403)
		return
	}

	err_delete_chirp := apiContext.queries.DeleteChirp(req.Context(), database.DeleteChirpParams{
		ID:     chirpUUID,
		UserID: userID,
	})

	if err_delete_chirp != nil {
		sendChirpError(w, "Chirp not found", 404)
		return
	}

	w.WriteHeader(204)
}

func handleChirpyRedSubscriptionPolkaWebhookHTTPRequest(w http.ResponseWriter, req *http.Request) {
	apiKey, err := auth.GetToken(req.Header, auth.ApiKey)
	if err != nil {
		sendChirpError(w, "invalid api key", 401)
		return
	}

	if validateAPIKey(apiKey) == false {
		sendChirpError(w, "invalid api key", 401)
		return
	}

	var chirpyRedRequestData PolkaChirpyRedWebhookRequest

	err_request_parse := json.NewDecoder(req.Body).Decode(&chirpyRedRequestData)
	if err_request_parse != nil {
		sendChirpError(w, "request format incorrect", 400)
		return
	}

	if chirpyRedRequestData.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	userUUID, err := uuid.Parse(chirpyRedRequestData.Data.UserID)
	if err != nil {
		sendChirpError(w, "Invalid user", 404)
		return
	}

	_, err_user_not_found := apiContext.queries.AddChirpyRed(req.Context(), userUUID)

	if err_user_not_found != nil {
		sendChirpError(w, "Invalid user", 404)
		return
	} else {
		w.WriteHeader(204)
	}
}

func validateAPIKey(apiKey string) bool {
	polkaAPIKey := os.Getenv("POLKA_KEY")
	return polkaAPIKey == apiKey
}

var apiContext *apiConfig

// func init() {
// 	apiContext = &apiConfig{}
// }

func main() {
	const testPort = "8082"
	const livePort = "8080"
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)

	var dbQueries *database.Queries
	var portToUse string
	apiContext = &apiConfig{}

	apiContext.platform = platform
	if err == nil {
		dbQueries = database.New(db)
		apiContext.queries = dbQueries
	}
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "live" {
			portToUse = livePort
		} else {
			portToUse = testPort
		}
	} else {
		portToUse = testPort
	}

	fmt.Println("--- RUNNING ABSOLUTE LOCAL VERSION !!!!! ---")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handleHealthzHTTPRequests)
	mux.HandleFunc("POST /api/users", handleCreateUserHTTPRequest)
	mux.HandleFunc("PUT /api/users", handleUpdateUserHTTPRequest)
	mux.HandleFunc("POST /api/refresh", handleRefreshTokenHTTPRequest)
	mux.HandleFunc("POST /api/revoke", handleRefreshTokenRevokeHTTPRequest)
	mux.HandleFunc("POST /api/login", handleLoginHTTPRequest)
	mux.HandleFunc("POST /api/polka/webhooks", handleChirpyRedSubscriptionPolkaWebhookHTTPRequest)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", handleDeleteChirpHTTPRequest)

	mux.HandleFunc("GET /admin/metrics", handleMetricsAndResetHTTPRequest)
	mux.HandleFunc("POST /api/chirps", handleAddChirpHTTPRequest)
	mux.HandleFunc("GET /api/chirps", handleGetAllChirpsHTTPRequest)
	mux.HandleFunc("GET /api/chirps/{chirpID}", handleGetChirpHTTPRequest)
	mux.Handle("POST /admin/reset", apiContext.middlewareMetricsInc(http.HandlerFunc(handleMetricsAndResetHTTPRequest)))

	fileServerHandler := http.FileServer(http.Dir("."))
	mux.Handle("/app/", apiContext.middlewareMetricsInc(http.StripPrefix("/app", fileServerHandler)))

	srv := &http.Server{
		Addr:    ":" + portToUse,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", portToUse)
	log.Fatal(srv.ListenAndServe())
}
