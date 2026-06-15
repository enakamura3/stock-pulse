package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerManager_RegisterAndGetAll(t *testing.T) {
	m := NewManager()

	job := func(ctx context.Context) {}
	w := NewWorker("TestWorker", 10*time.Millisecond, job)

	m.Register(w)

	infos := m.GetAll()
	if len(infos) != 1 {
		t.Fatalf("expected 1 worker info, got %d", len(infos))
	}

	if infos[0].Name != "TestWorker" {
		t.Errorf("expected worker name TestWorker, got %s", infos[0].Name)
	}
	if infos[0].Status != "idle" {
		t.Errorf("expected status idle, got %s", infos[0].Status)
	}
}

func TestWorkerManager_Trigger(t *testing.T) {
	m := NewManager()

	var counter int32
	job := func(ctx context.Context) {
		atomic.AddInt32(&counter, 1)
	}
	
	w := NewWorker("ManualWorker", 1*time.Hour, job)
	m.Register(w)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inicia os workers
	go m.StartAll(ctx)

	// Aguarda um momento para o Start inicial rodar
	time.Sleep(50 * time.Millisecond)

	// Primeira execução ocorre logo ao Start()
	if atomic.LoadInt32(&counter) != 1 {
		t.Errorf("expected counter to be 1 after start, got %d", atomic.LoadInt32(&counter))
	}

	// Faz um trigger manual
	ok := m.Trigger("ManualWorker")
	if !ok {
		t.Fatalf("expected trigger to succeed")
	}

	// Aguarda o job processar
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&counter) != 2 {
		t.Errorf("expected counter to be 2 after trigger, got %d", atomic.LoadInt32(&counter))
	}

	// Testa trigger num worker inexistente
	ok = m.Trigger("NonExistent")
	if ok {
		t.Errorf("expected trigger to fail for non-existent worker")
	}
}
