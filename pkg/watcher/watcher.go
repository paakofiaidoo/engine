package watcher

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	fsWatcher   *fsnotify.Watcher
	subscribers []chan fsnotify.Event
	errors      chan error
	done        chan bool
	debounce    time.Duration
	mu          sync.Mutex
	timers      map[string]*time.Timer
}

func NewWatcher(debounce time.Duration) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsWatcher:   fsWatcher,
		subscribers: make([]chan fsnotify.Event, 0),
		errors:      make(chan error),
		done:        make(chan bool),
		debounce:    debounce,
		timers:      make(map[string]*time.Timer),
	}, nil
}

func (w *Watcher) Add(path string) error {
	return w.fsWatcher.Add(path)
}

func (w *Watcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-w.fsWatcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					return
				}
				w.errors <- err
			}
		}
	}()
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if timer, exists := w.timers[event.Name]; exists {
		timer.Stop()
	}

	w.timers[event.Name] = time.AfterFunc(w.debounce, func() {
		w.broadcast(event)
		w.mu.Lock()
		delete(w.timers, event.Name)
		w.mu.Unlock()
	})
}

func (w *Watcher) broadcast(event fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, sub := range w.subscribers {
		// Non-blocking send to avoid blocking the watcher if a client is slow
		select {
		case sub <- event:
		default:
		}
	}
}

func (w *Watcher) Subscribe() <-chan fsnotify.Event {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch := make(chan fsnotify.Event, 10) // Buffer to prevent dropped events
	w.subscribers = append(w.subscribers, ch)
	return ch
}

func (w *Watcher) Close() error {
	return w.fsWatcher.Close()
}
