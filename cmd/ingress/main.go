package main

import (
	"log"
	"net/http"

	"github.com/popochoo/L7-ingress-controller/internal/config"
)

func main() {
	cfgManager := config.NewManager("config.yaml")

	if err := cfgManager.Load(); err != nil {
		log.Printf("Failed to load initial config: %v", err)
	}

	if err := cfgManager.Watch(); err != nil {
		log.Printf("Failed to start config watcher: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		currentCfg := cfgManager.Get()

		log.Printf("Handling request. Total routes configured: %d", len(currentCfg.Routes))
		w.Write([]byte("Ingress Gateway Active"))
	})

	serverCfg := cfgManager.Get().Server
	log.Printf("Starting Ingress Controller on %s", serverCfg.ListenAddr)

	server := &http.Server{Addr: serverCfg.ListenAddr}
	log.Fatal(server.ListenAndServe())
}
