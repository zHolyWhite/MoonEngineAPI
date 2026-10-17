package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Kind string

const (
	KindVoice  Kind = "voice"
	KindChat   Kind = "chat"
	KindManual Kind = "manual"
)

var ErrQueueFull = errors.New("queue_full")
var ErrExpired = errors.New("expired")
var ErrClosed = errors.New("closed")
var ErrInvalidTask = errors.New("invalid_task")

type Task struct {
	ID        string
	Kind      Kind
	SessionID string
	Deadline  time.Time
	Run       func(context.Context) error
}

type Config struct {
	VoiceQueueCap     int
	ChatQueueCap      int
	ManualQueueCap    int
	GlobalConcurrency int
	SessionMaxPending int
	VoiceTaskTTL      time.Duration
}

func LoadConfig() (Config, error) {
	cfg := Config{}
	var err error
	if cfg.VoiceQueueCap, err = getInt("MOONENGINE_VOICE_QUEUE_CAP", 32); err != nil {
		return cfg, err
	}
	if cfg.ChatQueueCap, err = getInt("MOONENGINE_CHAT_QUEUE_CAP", 64); err != nil {
		return cfg, err
	}
	if cfg.ManualQueueCap, err = getInt("MOONENGINE_MANUAL_QUEUE_CAP", 16); err != nil {
		return cfg, err
	}
	if cfg.GlobalConcurrency, err = getInt("MOONENGINE_GLOBAL_CONCURRENCY", 2); err != nil {
		return cfg, err
	}
	if cfg.SessionMaxPending, err = getInt("MOONENGINE_SESSION_MAX_PENDING", 4); err != nil {
		return cfg, err
	}
	ttlMs, err := getInt("MOONENGINE_VOICE_TASK_TTL_MS", 5000)
	if err != nil {
		return cfg, err
	}
	cfg.VoiceTaskTTL = time.Duration(ttlMs) * time.Millisecond
	return cfg, cfg.validate()
}

func (c Config) validate() error {
	if c.VoiceQueueCap <= 0 || c.ChatQueueCap <= 0 || c.ManualQueueCap <= 0 {
		return errors.New("las capacidades de cola deben ser mayores que 0")
	}
	if c.GlobalConcurrency <= 0 {
		return errors.New("MOONENGINE_GLOBAL_CONCURRENCY debe ser mayor que 0")
	}
	if c.SessionMaxPending <= 0 {
		return errors.New("MOONENGINE_SESSION_MAX_PENDING debe ser mayor que 0")
	}
	if c.VoiceTaskTTL <= 0 {
		return errors.New("MOONENGINE_VOICE_TASK_TTL_MS debe ser mayor que 0")
	}
	return nil
}

func getInt(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, errors.New(key + " no es un entero")
	}
	return n, nil
}

type Scheduler struct {
	cfg    Config
	log    *slog.Logger
	voice  chan *Task
	chat   chan *Task
	manual chan *Task
	slots  chan struct{}

	mu      sync.Mutex
	pending map[string]int64
	closed  bool
	done    chan struct{}

	enqueued  atomic.Int64
	processed atomic.Int64
	rejected  atomic.Int64
	expired   atomic.Int64
}

func New(cfg Config, log *slog.Logger) *Scheduler {
	return &Scheduler{
		cfg:     cfg,
		log:     log,
		voice:   make(chan *Task, cfg.VoiceQueueCap),
		chat:    make(chan *Task, cfg.ChatQueueCap),
		manual:  make(chan *Task, cfg.ManualQueueCap),
		slots:   make(chan struct{}, cfg.GlobalConcurrency),
		pending: make(map[string]int64),
		done:    make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	for i := 0; i < s.cfg.GlobalConcurrency; i++ {
		go s.worker(ctx)
	}
}

func (s *Scheduler) Enqueue(task *Task) error {
	if task == nil || task.Run == nil {
		return ErrInvalidTask
	}
	switch task.Kind {
	case KindVoice, KindChat, KindManual:
	default:
		return ErrInvalidTask
	}

	now := time.Now()
	if task.Kind == KindVoice {
		if task.Deadline.IsZero() {
			task.Deadline = now.Add(s.cfg.VoiceTaskTTL)
		}
		if now.After(task.Deadline) {
			s.expired.Add(1)
			return ErrExpired
		}
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	if s.pending[task.SessionID] >= int64(s.cfg.SessionMaxPending) {
		s.mu.Unlock()
		s.rejected.Add(1)
		return ErrQueueFull
	}
	s.pending[task.SessionID]++
	s.mu.Unlock()

	var ch chan *Task
	switch task.Kind {
	case KindVoice:
		ch = s.voice
	case KindChat:
		ch = s.chat
	default:
		ch = s.manual
	}

	select {
	case ch <- task:
		s.enqueued.Add(1)
		return nil
	default:
		s.mu.Lock()
		s.pending[task.SessionID]--
		s.mu.Unlock()
		s.rejected.Add(1)
		return ErrQueueFull
	}
}

func (s *Scheduler) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.done)
	s.mu.Unlock()
}

func (s *Scheduler) Stats() map[string]int64 {
	s.mu.Lock()
	pending := make(map[string]int64, len(s.pending))
	for k, v := range s.pending {
		pending[k] = v
	}
	s.mu.Unlock()
	return map[string]int64{
		"enqueued":  s.enqueued.Load(),
		"processed": s.processed.Load(),
		"rejected":  s.rejected.Load(),
		"expired":   s.expired.Load(),
		"pending":   int64(len(pending)),
	}
}

func (s *Scheduler) worker(ctx context.Context) {
	for {
		task, ok := s.receive()
		if !ok {
			return
		}
		s.process(ctx, task)
	}
}

func (s *Scheduler) receive() (*Task, bool) {
	select {
	case t := <-s.voice:
		return t, true
	default:
	}

	select {
	case t := <-s.chat:
		return t, true
	default:
	}

	select {
	case t := <-s.manual:
		return t, true
	default:
	}

	select {
	case t := <-s.voice:
		return t, true
	case t := <-s.chat:
		return t, true
	case t := <-s.manual:
		return t, true
	case <-s.done:
		return nil, false
	}
}

func (s *Scheduler) process(ctx context.Context, task *Task) {
	defer func() {
		s.mu.Lock()
		s.pending[task.SessionID]--
		s.mu.Unlock()
	}()

	defer func() {
		if rec := recover(); rec != nil {
			s.log.Error("panic en tarea", "taskId", task.ID, "kind", task.Kind, "panic", rec)
		}
	}()

	select {
	case s.slots <- struct{}{}:
	default:
		select {
		case s.slots <- struct{}{}:
		case <-s.done:
			return
		case <-ctx.Done():
			return
		}
	}
	defer func() { <-s.slots }()

	runCtx := ctx
	if !task.Deadline.IsZero() {
		remaining := time.Until(task.Deadline)
		if remaining <= 0 {
			s.expired.Add(1)
			return
		}
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, remaining)
		defer cancel()
	}

	if err := task.Run(runCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			s.expired.Add(1)
		} else {
			s.log.Warn("tarea falló", "taskId", task.ID, "kind", task.Kind, "error", err)
		}
		return
	}
	s.processed.Add(1)
}
