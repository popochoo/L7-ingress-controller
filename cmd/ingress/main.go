package main

import (
	"log"
	"net/http"

	"github.com/popochoo/L7-ingress-controller/internal/config"
	"github.com/popochoo/L7-ingress-controller/internal/metrics"
	"github.com/popochoo/L7-ingress-controller/internal/mirror"
	"github.com/popochoo/L7-ingress-controller/internal/proxy"
)

func main() {
	cfgManager := config.NewManager("config.yaml")
	if err := cfgManager.Load(); err != nil {
		log.Printf("Failed to load initial config: %v", err)
	}

	_ = cfgManager.Watch()

	metrics.InitMetrics()

	mirrorPool := mirror.NewWorkerPool(10, 100)
	mirrorPool.Start()

	proxyServer := proxy.NewProxyServer(cfgManager, mirrorPool)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.NewMetricsHandler())
		log.Println("Metrics server runnnig on :9090/metrics")
		_ = http.ListenAndServe(":9090", mux)
	}()

	serverCfg := cfgManager.Get().Server

	log.Printf("Ingress Controller listening on %s", serverCfg.ListenAddr)
	log.Fatal(http.ListenAndServe(serverCfg.ListenAddr, proxyServer))
}
