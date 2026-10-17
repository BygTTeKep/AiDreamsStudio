package claude

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrQueueFull     = errors.New("scenario generation queue is full")
	ErrWorkerStopped = errors.New("scenario generation worker is stopped")
)

// ScenarioSaver сохраняет результат генерации. Реализуется модулем scenario.
type ScenarioSaver interface {
	SaveGeneratedScenarios(ctx context.Context, job ScenarioJob, scenarios []GeneratedScenario) error
}

type Generator interface {
	GenerateScenarios(ctx context.Context, job ScenarioJob) ([]GeneratedScenario, error)
}

type WorkerConfig struct {
	Workers   int           // сколько запросов к Claude выполняется параллельно
	QueueSize int           // сколько задач может ждать в очереди
	Timeout   time.Duration // таймаут на одну задачу (генерация + сохранение)
}

// Worker — пул горутин, который забирает задачи из очереди,
// генерирует сценарии через Claude и отдаёт их в ScenarioSaver.
type Worker struct {
	gen     Generator
	saver   ScenarioSaver
	cfg     WorkerConfig
	jobs    chan ScenarioJob
	wg      sync.WaitGroup
	mu      sync.RWMutex
	stopped bool
}

func NewWorker(gen Generator, saver ScenarioSaver, cfg WorkerConfig) *Worker {
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Minute
	}
	return &Worker{
		gen:   gen,
		saver: saver,
		cfg:   cfg,
		jobs:  make(chan ScenarioJob, cfg.QueueSize),
	}
}

// SetSaver нужен, если saver создаётся после воркера (scenario.Service зависит от воркера).
func (w *Worker) SetSaver(saver ScenarioSaver) {
	w.saver = saver
}

// Start запускает обработчики. Они завершаются после Stop, дочитав очередь.
func (w *Worker) Start(ctx context.Context) {
	for i := 0; i < w.cfg.Workers; i++ {
		w.wg.Add(1)
		go func(id int) {
			defer w.wg.Done()
			for job := range w.jobs {
				w.process(ctx, id, job)
			}
		}(i)
	}
}

// Enqueue ставит задачу в очередь без блокировки.
func (w *Worker) Enqueue(job ScenarioJob) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.stopped {
		return ErrWorkerStopped
	}
	select {
	case w.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// Stop перестаёт принимать задачи и ждёт завершения уже принятых (или отмены ctx из Start).
func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		close(w.jobs)
	}
	w.mu.Unlock()
	w.wg.Wait()
}

func (w *Worker) process(ctx context.Context, id int, job ScenarioJob) {
	ctx, cancel := context.WithTimeout(ctx, w.cfg.Timeout)
	defer cancel()

	log := slog.With("worker", id, "project_id", job.ProjectID, "user_id", job.UserID)
	start := time.Now()

	scenarios, err := w.gen.GenerateScenarios(ctx, job)
	if err != nil {
		log.Error("claude scenario generation", "error", err, "duration", time.Since(start))
		return
	}
	if err := w.saver.SaveGeneratedScenarios(ctx, job, scenarios); err != nil {
		log.Error("save generated scenarios", "error", err)
		return
	}
	log.Info("scenarios generated", "count", len(scenarios), "duration", time.Since(start))
}
