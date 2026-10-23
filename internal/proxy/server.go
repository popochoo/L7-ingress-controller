package proxy

import (
	"bytes"
	"crypto/rand"
	"io"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/popochoo/L7-ingress-controller/internal/config"
	"github.com/popochoo/L7-ingress-controller/internal/metrics"
	"github.com/popochoo/L7-ingress-controller/internal/mirror"
	"github.com/popochoo/L7-ingress-controller/internal/router"
	"golang.org/x/time/rate"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

type ProxyServer struct {
	configManager interface{ Get() *config.Config }
	mirrorPool    *mirror.WorkerPool
	limiters      map[string]*rate.Limiter
	mu            sync.RWMutex
	client        *http.Client
}

func NewProxyServer(
	cm interface{ Get() *config.Config },
	mp *mirror.WorkerPool,
) *ProxyServer {
	return &ProxyServer{
		configManager: cm,
		mirrorPool:    mp,
		limiters:      make(map[string]*rate.Limiter),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (ps *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	currentConfig := ps.configManager.Get()
	rt := router.NewRouter(currentConfig)

	route, found := rt.Match(r)
	if !found {
		http.Error(w, "Not Found (Ingress)", http.StatusNotFound)
		return
	}

	lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	defer func() {
		metrics.GetMetrics().ObserveDuration(route.Path, r.Method, startTime)
		metrics.GetMetrics().IncRequests(route.Path, r.Method, lrw.statusCode)
	}()

	if route.RateLimitRPS > 0 {
		limiter := ps.getLimiter(route.Path, route.RateLimitRPS)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests (Rate Limit)", http.StatusTooManyRequests)
			return
		}
	}

	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request Body", http.StatusBadRequest)
			return
		}
	}

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if route.Mirror != nil && route.Mirror.Percentage > 0 {
		n, err := rand.Int(rand.Reader, big.NewInt(100))
		if err == nil && int(n.Int64()) < route.Mirror.Percentage {
			mURL, err := url.Parse(route.Mirror.URL)
			if err == nil {
				task := mirror.Task{
					TargetURL: mURL,
					Method:    r.Method,
					Path:      r.URL.Path,
					Header:    r.Header,
					Body:      bodyBytes,
				}

				if ps.mirrorPool.TryQueue(task) {
					metrics.GetMetrics().IncMirrored(route.Path)
				} else {
					metrics.GetMetrics().IncDropped(route.Path)
				}
			}
		}
	}

	ps.forwardWithFallback(lrw, r, route)
}

func (ps *ProxyServer) forwardWithFallback(w http.ResponseWriter, r *http.Request, route config.RouteConfig) {
	prodURL, err := url.Parse(route.Backend.URL)
	if err != nil {
		http.Error(w, "Bad Gateway Config", http.StatusBadGateway)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(prodURL)

	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, proxyErr error) {
		if route.Backend.FallbackURL != "" {
			fallbackURL, err := url.Parse(route.Backend.FallbackURL)
			if err == nil {
				fallbackProxy := httputil.NewSingleHostReverseProxy(fallbackURL)
				fallbackProxy.ServeHTTP(w, req)
				return
			}
		}

		http.Error(w, "Service Unavailable (Production & Fallback Failed)", http.StatusServiceUnavailable)
	}

	proxy.ServeHTTP(w, r)
}

func (ps *ProxyServer) getLimiter(path string, rps int) *rate.Limiter {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	limiter, exists := ps.limiters[path]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rps), rps)
		ps.limiters[path] = limiter
	}

	return limiter
}
