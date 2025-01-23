package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
)

type User struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

var userCache = make(map[int]User)

var cacheMutex sync.RWMutex

func main() {
	mux := http.NewServeMux()
	// mux.HandleFunc("/", handleRoot)

	mux.HandleFunc("POST /user", getUserAuth)
	mux.HandleFunc("GET /user/{id}", getUser)
	mux.HandleFunc("DELETE /user/{id}", deleteUser)

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", mux)
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

func getUserAuth(writer http.ResponseWriter, request *http.Request) {
	var user User

	err := json.NewDecoder(request.Body).Decode((&user))

	handleError(writer, err, http.StatusBadRequest)

	fmt.Printf("handle: %+v\n", user.Identifier)
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

	fmt.Printf("resp Body: %v\n", string(body))

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
