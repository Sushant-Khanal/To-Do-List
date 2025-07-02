package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Task represents a todo item
type Task struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskStore manages tasks in memory (you could replace this with a database)
type TaskStore struct {
	tasks  map[int]*Task
	nextID int
	mu     sync.RWMutex
}

// NewTaskStore creates a new task store
func NewTaskStore() *TaskStore {
	store := &TaskStore{
		tasks:  make(map[int]*Task),
		nextID: 1,
	}

	// Add some default tasks
	store.addDefaultTasks()
	return store
}

func (ts *TaskStore) addDefaultTasks() {
	defaultTasks := []string{
		"Watch Go course",
		"Learn Distributed databases",
		"Take some rest and eat some snacks",
	}

	for _, text := range defaultTasks {
		ts.CreateTask(text)
	}
}

// CreateTask adds a new task
func (ts *TaskStore) CreateTask(text string) *Task {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	task := &Task{
		ID:        ts.nextID,
		Text:      strings.TrimSpace(text),
		Completed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ts.tasks[ts.nextID] = task
	ts.nextID++

	return task
}

// GetAllTasks returns all tasks
func (ts *TaskStore) GetAllTasks() []*Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	tasks := make([]*Task, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTask returns a task by ID
func (ts *TaskStore) GetTask(id int) (*Task, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	task, exists := ts.tasks[id]
	return task, exists
}

// UpdateTask updates a task's completion status
func (ts *TaskStore) UpdateTask(id int, completed bool) (*Task, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	task, exists := ts.tasks[id]
	if !exists {
		return nil, false
	}

	task.Completed = completed
	task.UpdatedAt = time.Now()

	return task, true
}

// DeleteTask removes a task
func (ts *TaskStore) DeleteTask(id int) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	_, exists := ts.tasks[id]
	if exists {
		delete(ts.tasks, id)
	}

	return exists
}

// GetStats returns task statistics
func (ts *TaskStore) GetStats() map[string]int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	total := len(ts.tasks)
	completed := 0

	for _, task := range ts.tasks {
		if task.Completed {
			completed++
		}
	}

	return map[string]int{
		"total":     total,
		"completed": completed,
		"pending":   total - completed,
	}
}

// Server handles HTTP requests
type Server struct {
	store *TaskStore
}

// NewServer creates a new server
func NewServer() *Server {
	return &Server{
		store: NewTaskStore(),
	}
}

// API Handlers

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getTasks(w, r)
	case http.MethodPost:
		s.createTask(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.store.GetAllTasks()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, "Error encoding tasks", http.StatusInternalServerError)
		return
	}
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		http.Error(w, "Task text cannot be empty", http.StatusBadRequest)
		return
	}

	task := s.store.CreateTask(req.Text)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.updateTask(w, r, id)
	case http.MethodDelete:
		s.deleteTask(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request, id int) {
	var req struct {
		Completed bool `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	task, exists := s.store.UpdateTask(id, req.Completed)
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request, id int) {
	if !s.store.DeleteTask(id) {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.store.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Middleware for logging requests
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Middleware for CORS
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

var startTime = time.Now()

func main() {
	server := NewServer()

	// Serve static files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
		} else {
			http.NotFound(w, r)
		}
	})

	// API routes with middleware
	http.HandleFunc("/api/tasks", corsMiddleware(loggingMiddleware(server.handleTasks)))
	http.HandleFunc("/api/tasks/", corsMiddleware(loggingMiddleware(server.handleTaskByID)))
	http.HandleFunc("/api/stats", corsMiddleware(loggingMiddleware(server.handleStats)))
	http.HandleFunc("/api/health", corsMiddleware(loggingMiddleware(server.handleHealth)))

	port := ":8080"
	log.Printf("Starting Todo server on port %s", port)
	log.Printf("Open http://localhost%s in your browser", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
