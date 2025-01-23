package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type User struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type OauthMetadata struct {
	ClientName              string    `json:"client_name"`
	ClientId                string    `json:"client_id"`
	ClientUri               string    `json:"client_uri"`
	RedirectUris            [1]string `json:"redirect_uris"`
	Scope                   string    `json:"scope"`
	GrantTypes              [2]string `json:"grant_types"`
	ResponseTypes           [1]string `json:"response_types"`
	ApplicationType         string    `json:"application_type"`
	TokenEndpointAuthMethod string    `json:"token_endpoint_auth_method"`
	DpopBoundAccessToken    bool      `json:"dpop_bound_access_tokens"`
}

var userCache = make(map[int]User)

var cacheMutex sync.RWMutex

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	mux := http.NewServeMux()
	// mux.HandleFunc("/", handleRoot)

	mux.HandleFunc("GET /client-metadata.json", exposeMetadata)
	mux.HandleFunc("GET /user/{id}", getUser)
	mux.HandleFunc("POST /user", getUserAuth)
	mux.HandleFunc("DELETE /user/{id}", deleteUser)

	appPort := os.Getenv("PORT")
	fmt.Println("Listening on port", appPort)

	addr := fmt.Sprintf(":%s", appPort)
	http.ListenAndServe(addr, mux)
}

// func handleRoot(
// 	writer http.ResponseWriter,
// 	request *http.Request,
// ) {
// 	fmt.Fprintf(writer, "Hello world")
// }

func handleError(writer http.ResponseWriter, err error, status int) {
	if err != nil {
		http.Error(
			writer,
			err.Error(),
			status,
		)

		return
	}
}

func isValidHandle(handle string) bool {
	pattern := "[a-zA-Z0-9.-]"
	match, _ := regexp.MatchString(pattern, handle)

	return match
}

func exposeMetadata(writer http.ResponseWriter, request *http.Request) {
	metadata := OauthMetadata{
		"Social data Transfer",
		"localhost:8080/client-metadata.json",
		"localhost:8080",
		[1]string{"localhost:8080/oauth/callback"},
		"atproto transition:generic",
		[2]string{"authorization_code", "refresh_token"},
		[1]string{"code"},
		"web",
		"none",
		true,
	}

	metadataJson, _ := json.Marshal(metadata)

	writer.WriteHeader(http.StatusOK)
	writer.Write(metadataJson)
}

func getUserAuth(writer http.ResponseWriter, request *http.Request) {
	var user User

	err := json.NewDecoder(request.Body).Decode((&user))
	handleError(writer, err, http.StatusBadRequest)

	if !isValidHandle(user.Identifier) {
		http.Error(
			writer,
			"Invalid handle",
			http.StatusBadRequest,
		)

		return
	}

	bodyJSON, _ := json.Marshal(user)

	req, err := http.NewRequest("POST", "https://bsky.social/xrpc/com.atproto.server.createSession", bytes.NewBuffer(bodyJSON))

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	handleError(writer, err, http.StatusInternalServerError)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(body)
}

func createUser(writer http.ResponseWriter, request *http.Request) {
	var user User

	err := json.NewDecoder(request.Body).Decode(&user)

	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	if user.Identifier == "" {
		http.Error(
			writer,
			"Name is required",
			http.StatusBadRequest,
		)

		return
	}

	cacheMutex.Lock()
	userCache[len(userCache)+1] = user
	cacheMutex.Unlock()

	writer.WriteHeader((http.StatusNoContent))
}

func deleteUser(writer http.ResponseWriter, request *http.Request) {
	id, err := strconv.Atoi(request.PathValue("id"))

	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	if _, ok := userCache[id]; !ok {
		http.Error(
			writer,
			"user not found",
			http.StatusBadRequest,
		)

		return
	}

	cacheMutex.Lock()
	delete(userCache, id)
	cacheMutex.Unlock()

	writer.WriteHeader(http.StatusNoContent)
}

func getUser(writer http.ResponseWriter, request *http.Request) {
	id, err := strconv.Atoi(request.PathValue("id"))

	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	cacheMutex.RLock()
	user, ok := userCache[id]
	cacheMutex.RUnlock()

	if !ok {
		http.Error(
			writer,
			"user not found",
			http.StatusNotFound,
		)
	}

	writer.Header().Set("Content-Type", "application/json")

	j, err := json.Marshal(user)

	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusInternalServerError,
		)
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write(j)
}
