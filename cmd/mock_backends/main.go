package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "source": "production"}`))
		})
		log.Println("Production mock running on :8081")
		log.Fatal(http.ListenAndServe(":8081", mux))
	}()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "source": "test_mirror"}`))
		})
		log.Println("Slow Test/ML mock running on :8082")
		log.Fatal(http.ListenAndServe(":8082", mux))
	}()

	select {}
}
