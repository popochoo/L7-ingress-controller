package mirror

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Task struct {
	TargetURL *url.URL
	Method    string
	Path      string
	Header    http.Header
	Body      []byte
}

type WorkerPool struct {
	queue   chan Task
	client  *http.Client
	workers int
}

func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	return &WorkerPool{
		queue:   make(chan Task, queueSize),
		workers: workers,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		go p.worker()
	}
}

func (p *WorkerPool) worker() {
	for task := range p.queue {
		p.sendMirror(task)
	}
}

func (p *WorkerPool) TryQueue(task Task) bool {
	select {
	case p.queue <- task:
		return true

	default:
		return false
	}
}

func (p *WorkerPool) sendMirror(task Task) {
	u := *task.TargetURL
	u.Path = task.Path

	req, err := http.NewRequestWithContext(context.Background(), task.Method, u.String(), bytes.NewReader(task.Body))
	if err != nil {
		return
	}

	req.Header = task.Header.Clone()

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[Mirror Worker] Error sending mirrored request: %v", err)
		return
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)
}
