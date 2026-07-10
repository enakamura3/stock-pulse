package worker

import (
	"context"
	"sync"
	"time"
)

type Info struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	LastRun     *time.Time `json:"last_run"`
	NextRun     *time.Time `json:"next_run"`
	Status      string     `json:"status"`
	Interval    string     `json:"interval"`
}

type Job func(ctx context.Context)

type Worker struct {
	Name        string
	Description string
	Interval    time.Duration
	Job         Job

	lastRun  time.Time
	nextRun  time.Time
	status   string
	mu       sync.Mutex
	trigger  chan struct{}
}

func NewWorker(name string, description string, interval time.Duration, job Job) *Worker {
	return &Worker{
		Name:        name,
		Description: description,
		Interval:    interval,
		Job:         job,
		status:      "idle",
		trigger:     make(chan struct{}, 1),
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.mu.Lock()
	w.nextRun = time.Now()
	w.mu.Unlock()

	w.execute(ctx)

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.execute(ctx)
		case <-w.trigger:
			w.execute(ctx)
			ticker.Reset(w.Interval)
		}
	}
}

func (w *Worker) execute(ctx context.Context) {
	w.mu.Lock()
	w.status = "running"
	w.lastRun = time.Now()
	w.mu.Unlock()

	w.Job(ctx)

	w.mu.Lock()
	w.status = "idle"
	w.nextRun = time.Now().Add(w.Interval)
	w.mu.Unlock()
}

func (w *Worker) Trigger() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *Worker) Info() Info {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	var lr, nr *time.Time
	if !w.lastRun.IsZero() {
		last := w.lastRun
		lr = &last
	}
	if !w.nextRun.IsZero() {
		next := w.nextRun
		nr = &next
	}

	return Info{
		Name:        w.Name,
		Description: w.Description,
		LastRun:     lr,
		NextRun:     nr,
		Status:      w.status,
		Interval:    w.Interval.String(),
	}
}

type Manager struct {
	workers map[string]*Worker
	mu      sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		workers: make(map[string]*Worker),
	}
}

func (m *Manager) Register(w *Worker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workers[w.Name] = w
}

func (m *Manager) StartAll(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.workers {
		go w.Start(ctx)
	}
}

func (m *Manager) GetAll() []Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var infos []Info
	for _, w := range m.workers {
		infos = append(infos, w.Info())
	}
	return infos
}

func (m *Manager) Trigger(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.workers[name]
	if !ok {
		return false
	}
	w.Trigger()
	return true
}
