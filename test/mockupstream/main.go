package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func getMockResponse(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Has("after") {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
		return
	}

	data, err := os.ReadFile(filepath.Join("/", fmt.Sprintf("%s.json", query.Get("domain"))))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func main() {
	http.HandleFunc("/", getMockResponse)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
