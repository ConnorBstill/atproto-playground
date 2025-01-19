package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type User struct {
	Name string `json:"name"`
}

var userCache = make(map[int]User)

var cacheMutex sync.RWMutex

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)

	mux.HandleFunc("POST /user", createUser)
	mux.HandleFunc("GET /user/{id}", getUser)
	mux.HandleFunc("DELETE /user/{id}", deleteUser)

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", mux)
}

func handleRoot(
	writer http.ResponseWriter,
	request *http.Request,
) {
	fmt.Fprintf(writer, "Hello world")
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

	if user.Name == "" {
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
