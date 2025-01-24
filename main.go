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

	oauth "atproto-playground/utils"

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
	mux.HandleFunc("POST /initiate-oauth", initiateOauth)
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

func initiateOauth(writer http.ResponseWriter, request *http.Request) {
	// oauth.GenerateVerifier(64)
	oauth.GeneratePKCE(64)
	// https://bsky.social/oauth/authorize?client_id=http%3A%2F%2Flocalhost%3Fredirect_uri%3Dhttp%253A%252F%252F127.0.0.1%253A8080%252Foauth%252Fcallback%26scope%3Datproto%2520transition%253Ageneric&request_uri=urn%3Aietf%3Aparams%3Aoauth%3Arequest_uri%3Areq-e14975c48319643639a8a4f743b263e2

	writer.Header().Set("Content-Type", "application/json")
	response, _ := json.Marshal(("Okay"))
	writer.Write(response)
}

func exposeMetadata(writer http.ResponseWriter, request *http.Request) {
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	hostUrl := port + host

	metadata := OauthMetadata{
		ClientName:              "Social data Transfer",
		ClientId:                hostUrl + "/client-metadata.json",
		ClientUri:               hostUrl,
		RedirectUris:            [1]string{hostUrl + "/oauth/callback"},
		Scope:                   "atproto transition:generic",
		GrantTypes:              [2]string{"authorization_code", "refresh_token"},
		ResponseTypes:           [1]string{"code"},
		ApplicationType:         "web",
		TokenEndpointAuthMethod: "none",
		DpopBoundAccessToken:    true,
	}

	metadataJson, _ := json.Marshal(metadata)

	writer.WriteHeader(http.StatusOK)
	writer.Write(metadataJson)
}

func getUserAuth(writer http.ResponseWriter, request *http.Request) {
	var user User

	err := json.NewDecoder(request.Body).Decode(&user)
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
