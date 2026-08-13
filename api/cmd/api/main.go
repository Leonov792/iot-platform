package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// порт по умолчанию, если окружение не настроено
const defaultPort = "8080"

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		// TODO: вынести чтение конфига в отдельный файл, надоело по мелочи лазить
		port = defaultPort
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// пока единственный "рут", потом тут будет версионированный api
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "iot api alive")
	})

	addr := ":" + port
	log.Printf("поднимаю http на %s", addr)

	// без таймаутов пока, в проде обязательно добавить
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("сервер сдох: %v", err)
	}
}
