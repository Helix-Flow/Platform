package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// GPUOptimizer handles advanced GPU resource management and optimization
type GPUOptimizer struct {
	gpus           map[string]*GPUDevice
	modelCache     *ModelCache
	scheduler      *GPUScheduler
	memoryManager  *GPUMemoryManager
	performanceMonitor *GPUPerformanceMonitor
	
	config         *GPUOptimizationConfig
	mutex          sync.RWMutex
}

// GPUOptimizationConfig holds GPU optimization settings
type GPUOptimizationConfig struct {
	// Memory management
	MemoryFraction      float64       `json:"memory_fraction"`
	MemoryReservation   uint64        `json:"memory_reservation"`
	MemoryDefragEnabled bool          `json:"memory_defrag_enabled"`
	
	// Model management
	ModelCacheEnabled   bool          `json:"model_cache_enabled"`
	ModelPreloadEnabled bool          `json:"model_preload_enabled"`
	ModelSwapEnabled    bool          `json:"model_swap_enabled"`
	
	// Performance optimization
	DynamicScheduling   bool          `json:"dynamic_scheduling"`
	LoadBalancing       bool          `json:"load_balancing"`
	PredictiveScaling   bool          `json:"predictive_scaling"`
	
	// Monitoring
	MetricsEnabled      bool          `json:"metrics_enabled"`
	ProfilingEnabled    bool          `json:"profiling_enabled"`
	
	// Advanced features
	MultiGPUEnabled     bool          `json:"multi_gpu_enabled"`
	TensorParallelism   bool          `json:"tensor_parallelism"`
	PipelineParallelism bool          `json:"pipeline_parallelism"`
}

// GPUDevice represents a physical GPU device
type GPUDevice struct {
	ID              string
	Name            string
	MemoryTotal     uint64
	MemoryUsed      uint64
	MemoryAvailable uint64
	Utilization     float64
	Temperature     float64
	PowerUsage      float64
	
	// Status
	Status          string // "available", "busy", "error"
	LastUpdate      time.Time
	
	// Assigned models
	ActiveModels    map[string]*ModelAllocation
	PendingModels   []*ModelAllocation
	
	// Performance metrics
	AvgLatency      time.Duration
	Throughput      float64
	ErrorRate       float64
}

// ModelAllocation represents a model allocated to a GPU
type ModelAllocation struct {
	ModelID         string
	ModelName       string
	MemoryRequired  uint64
	MemoryUsed      uint64
	LoadTime        time.Duration
	InferenceCount  int64
	LastAccessed    time.Time
	
	// Performance
	AvgLatency      time.Duration
	Throughput      float64
	ErrorCount      int64
}

// GPUScheduler handles intelligent GPU task scheduling
type GPUScheduler struct {
	queue      chan *SchedulingRequest
	workers    int
	algorithms map[string]SchedulingAlgorithm
}

// SchedulingRequest represents a request for GPU resources
type SchedulingRequest struct {
	ModelID      string
	ModelName    string
	MemoryRequired uint64
	Priority     int
	UserID       string
	Deadline     time.Time
	
	// Callbacks
	OnScheduled  func(gpuID string) error
	OnCompleted  func() error
	OnFailed     func(error) error
}

// SchedulingAlgorithm interface for different scheduling strategies
type SchedulingAlgorithm interface {
	SelectGPU(request *SchedulingRequest, availableGPUs []*GPUDevice) (string, error)
	Name() string
}

// GPUMemoryManager handles advanced GPU memory management
type GPUMemoryManager struct {
	allocator   *MemoryAllocator
	defragmenter *MemoryDefragmenter
	reservation *MemoryReservation
}

// MemoryAllocator handles memory allocation strategies
type MemoryAllocator struct {
	strategy string // "best_fit", "first_fit", "worst_fit"
}

// MemoryDefragmenter handles memory defragmentation
type MemoryDefragmenter struct {
	threshold float64
	interval  time.Duration
}

// MemoryReservation handles memory reservation for critical operations
type MemoryReservation struct {
	reservations map[string]*Reservation
	mutex        sync.RWMutex
}

// Reservation represents a memory reservation
type Reservation struct {
	ID          string
	GPUID       string
	MemorySize  uint64
	Purpose     string
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

// GPUPerformanceMonitor tracks GPU performance metrics
type GPUPerformanceMonitor struct {
	metrics map[string]*GPUMetrics
	mutex   sync.RWMutex
}

// GPUMetrics represents GPU performance metrics
type GPUMetrics struct {
	GPUID           string
	MemoryUsage     float64
	Utilization     float64
	Temperature     float64
	PowerUsage      float64
	InferenceCount  int64
	AvgLatency      time.Duration
	Throughput      float64
	ErrorRate       float64
	
	// Historical data
	LastUpdated     time.Time
	History         []MetricPoint
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Timestamp time.Time
	Value     float64
	Label     string
}

// NewGPUOptimizer creates a new GPU optimizer
func NewGPUOptimizer(config *GPUOptimizationConfig) *GPUOptimizer {
	optimizer := &GPUOptimizer{
		gpus:           make(map[string]*GPUDevice),
		modelCache:     NewModelCache(),
		performanceMonitor: NewGPUPerformanceMonitor(),
		config:         config,
	}
	
	// Initialize components
	optimizer.scheduler = NewGPUScheduler(config)
	optimizer.memoryManager = NewGPUMemoryManager(config)
	
	// Initialize with mock GPUs (in real implementation, detect actual GPUs)
	optimizer.initializeGPUs()
	
	// Start background tasks
	go optimizer.startBackgroundTasks()
	
	return optimizer
}

// initializeGPUs sets up GPU devices (mock implementation)
func (go *GPUOptimizer) initializeGPUs() {
	// Mock GPU initialization - in real implementation, detect actual GPUs
	gpuConfigs := []struct {
		id       string
		name     string
		memory   uint64
	}{
		{"gpu_0", "NVIDIA A100 40GB", 40 * 1024 * 1024 * 1024},
		{"gpu_1", "NVIDIA A100 40GB", 40 * 1024 * 1024 * 1024},
		{"gpu_2", "NVIDIA RTX 4090 24GB", 24 * 1024 * 1024 * 1024},
		{"gpu_3", "NVIDIA RTX 4090 24GB", 24 * 1024 * 1024 * 1024},
	}
	
	for _, config := range gpuConfigs {
		gpu := &GPUDevice{
			ID:              config.id,
			Name:            config.name,
			MemoryTotal:     config.memory,
			MemoryAvailable: config.memory,
			MemoryUsed:      0,
			Utilization:     0.0,
			Temperature:     35.0,
			PowerUsage:      250.0,
			Status:          "available",
			LastUpdate:      time.Now(),
			ActiveModels:    make(map[string]*ModelAllocation),
			PendingModels:   make([]*ModelAllocation, 0),
			AvgLatency:      50 * time.Millisecond,
			Throughput:      100.0,
			ErrorRate:       0.01,
		}
		
		go.gpus[config.id] = gpu
	}
}

// AllocateGPU intelligently allocates GPU resources
func (go *GPUOptimizer) AllocateGPU(ctx context.Context, request *SchedulingRequest) (string, error) {
	go.mutex.Lock()
	defer go.mutex.Unlock()
	
	log.Printf("Allocating GPU for model %s (memory: %d bytes)", request.ModelName, request.MemoryRequired)
	
	// Get available GPUs
	availableGPUs := go.getAvailableGPUs(request.MemoryRequired)
	if len(availableGPUs) == 0 {
		return "", fmt.Errorf("no GPUs available with sufficient memory for model %s", request.ModelName)
	}
	
	// Select best GPU using scheduling algorithm
	selectedGPU, err := go.scheduler.SelectGPU(request, availableGPUs)
	if err != nil {
		return "", fmt.Errorf("failed to select GPU: %w", err)
	}
	
	// Create model allocation
	allocation := &ModelAllocation{
		ModelID:        request.ModelID,
		ModelName:      request.ModelName,
		MemoryRequired: request.MemoryRequired,
		MemoryUsed:     request.MemoryRequired,
		LoadTime:       time.Now().Sub(time.Now()), // Will be updated
		InferenceCount: 0,
		LastAccessed:   time.Now(),
		AvgLatency:     0,
		Throughput:     0,
		ErrorCount:     0,
	}
	
	// Update GPU state
	gpu := go.gpus[selectedGPU]
	gpu.MemoryUsed += request.MemoryRequired
	gpu.MemoryAvailable -= request.MemoryRequired
	gpu.ActiveModels[request.ModelID] = allocation
	gpu.Status = "busy"
	gpu.LastUpdate = time.Now()
	
	log.Printf("Allocated GPU %s for model %s", selectedGPU, request.ModelName)
	
	// Trigger callback
	if request.OnScheduled != nil {
		if err := request.OnScheduled(selectedGPU); err != nil {
			return "", fmt.Errorf("scheduled callback failed: %w", err)
		}
	}
	
	return selectedGPU, nil
}

// ReleaseGPU releases GPU resources
func (go *GPUOptimizer) ReleaseGPU(ctx context.Context, gpuID string, modelID string) error {
	go.mutex.Lock()
	defer go.mutex.Unlock()
	
	gpu, exists := go.gpus[gpuID]
	if !exists {
		return fmt.Errorf("GPU %s not found", gpuID)
	}
	
	allocation, exists := gpu.ActiveModels[modelID]
	if !exists {
		return fmt.Errorf("model %s not found on GPU %s", modelID, gpuID)
	}
	
	log.Printf("Releasing GPU %s for model %s", gpuID, modelID)
	
	// Update GPU state
	gpu.MemoryUsed -= allocation.MemoryUsed
	gpu.MemoryAvailable += allocation.MemoryUsed
	delete(gpu.ActiveModels, modelID)
	
	// Update GPU status
	if len(gpu.ActiveModels) == 0 {
		gpu.Status = "available"
	}
	
	gpu.LastUpdate = time.Now()
	
	log.Printf("Released GPU %s for model %s", gpuID, modelID)
	
	return nil
}

// OptimizeMemory performs memory optimization
func (go *GPUOptimizer) OptimizeMemory(ctx context.Context) error {
	go.mutex.Lock()
	defer go.mutex.Unlock()
	
	print_status "Starting memory optimization..."
	
	// Defragment memory if enabled
	if go.config.MemoryDefragEnabled {
		if err := go.memoryManager.defragmenter.Defragment(go.gpus); err != nil {
			log.Printf("Memory defragmentation failed: %v", err)
		}
	}
	
	// Evict unused models
	if err := go.evictUnusedModels(ctx); err != nil {
		return fmt.Errorf("failed to evict unused models: %w", err)
	}
	
	// Preload frequently used models
	if go.config.ModelPreloadEnabled {
		if err := go.preloadFrequentModels(ctx); err != nil {
			return fmt.Errorf("failed to preload models: %w", err)
		}
	}
	
	print_status "Memory optimization completed"
	return nil
}

// evictUnusedModels removes models that haven't been used recently
func (go *GPUOptimizer) evictUnusedModels(ctx context.Context) error {
	threshold := 30 * time.Minute // Models unused for 30 minutes
	
	for _, gpu := range go.gpus {
		modelsToEvict := make([]string, 0)
		
		for modelID, allocation := range gpu.ActiveModels {
			if time.Since(allocation.LastAccessed) > threshold {
				modelsToEvict = append(modelsToEvict, modelID)
			}
		}
		
		for _, modelID := range modelsToEvict {
			if err := go.ReleaseGPU(ctx, gpu.ID, modelID); err != nil {
				log.Printf("Failed to evict model %s from GPU %s: %v", modelID, gpu.ID, err)
			}
		}
	}
	
	return nil
}

// preloadFrequentModels preloads models that are frequently requested
func (go *GPUOptimizer) preloadFrequentModels(ctx context.Context) error {
	// Analyze usage patterns and identify frequently used models
	frequentModels := go.analyzeUsagePatterns()
	
	for _, modelInfo := range frequentModels {
		// Check if model is already loaded
		alreadyLoaded := false
		for _, gpu := range go.gpus {
			if _, exists := gpu.ActiveModels[modelInfo.ModelID]; exists {
				alreadyLoaded = true
				break
			}
		}
		
		if !alreadyLoaded {
			// Find available GPU
			availableGPUs := go.getAvailableGPUs(modelInfo.MemoryRequired)
			if len(availableGPUs) > 0 {
				// Preload model to first available GPU
				request := &SchedulingRequest{
					ModelID:        modelInfo.ModelID,
					ModelName:      modelInfo.ModelName,
					MemoryRequired: modelInfo.MemoryRequired,
					Priority:       1, // Low priority for preloading
					UserID:         "system",
					Deadline:       time.Now().Add(5 * time.Minute),
				}
				
				if _, err := go.AllocateGPU(ctx, request); err != nil {
					log.Printf("Failed to preload model %s: %v", modelInfo.ModelID, err)
				}
			}
		}
	}
	
	return nil
}

// analyzeUsagePatterns analyzes GPU usage patterns
func (go *GPUOptimizer) analyzeUsagePatterns() []*ModelUsageInfo {
	usageMap := make(map[string]*ModelUsageInfo)
	
	for _, gpu := range go.gpus {
		for modelID, allocation := range gpu.ActiveModels {
			if usage, exists := usageMap[modelID]; exists {
				usage.RequestCount += allocation.InferenceCount
				usage.LastAccessed = maxTime(usage.LastAccessed, allocation.LastAccessed)
			} else {
				usageMap[modelID] = &ModelUsageInfo{
					ModelID:        modelID,
					ModelName:      allocation.ModelName,
					MemoryRequired: allocation.MemoryRequired,
					RequestCount:   allocation.InferenceCount,
					LastAccessed:   allocation.LastAccessed,
				}
			}
		}
	}
	
	// Convert to slice and sort by usage
	var usages []*ModelUsageInfo
	for _, usage := range usageMap {
		usages = append(usages, usage)
	}
	
	// Sort by request count (descending)
	for i := 0; i < len(usages)-1; i++ {
		for j := i + 1; j < len(usages); j++ {
			if usages[i].RequestCount < usages[j].RequestCount {
				usages[i], usages[j] = usages[j], usages[i]
			}
		}
	}
	
	return usages
}

// getAvailableGPUs returns GPUs with sufficient memory
func (go *GPUOptimizer) getAvailableGPUs(memoryRequired uint64) []*GPUDevice {
	var available []*GPUDevice
	
	for _, gpu := range go.gpus {
		if gpu.Status == "available" && gpu.MemoryAvailable >= memoryRequired {
			available = append(available, gpu)
		}
	}
	
	return available
}

// startBackgroundTasks starts background optimization tasks
func (go *GPUOptimizer) startBackgroundTasks() {
	// Memory optimization every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		if err := go.OptimizeMemory(ctx); err != nil {
			log.Printf("Background memory optimization failed: %v", err)
		}
	}
}

// GetMetrics returns GPU performance metrics
func (go *GPUOptimizer) GetMetrics() map[string]interface{} {
	go.mutex.RLock()
	defer go.mutex.RUnlock()
	
	metrics := make(map[string]interface{})
	
	// Overall metrics
	totalGPUs := len(go.gpus)
	availableGPUs := 0
	busyGPUs := 0
	totalMemory := uint64(0)
	usedMemory := uint64(0)
	
	for _, gpu := range go.gpus {
		totalMemory += gpu.MemoryTotal
		usedMemory += gpu.MemoryUsed
		
		switch gpu.Status {
		case "available":
			availableGPUs++
		case "busy":
			busyGPUs++
		}
	}
	
	metrics["total_gpus"] = totalGPUs
	metrics["available_gpus"] = availableGPUs
	metrics["busy_gpus"] = busyGPUs
	metrics["total_memory_bytes"] = totalMemory
	metrics["used_memory_bytes"] = usedMemory
	metrics["memory_utilization"] = float64(usedMemory) / float64(totalMemory) * 100
	
	// Per-GPU metrics
	gpuMetrics := make([]map[string]interface{}, 0)
	for _, gpu := range go.gpus {
		gpuMetric := map[string]interface{}{
			"id":              gpu.ID,
			"name":            gpu.Name,
			"status":          gpu.Status,
			"memory_total":    gpu.MemoryTotal,
			"memory_used":     gpu.MemoryUsed,
			"memory_available": gpu.MemoryAvailable,
			"utilization":     gpu.Utilization,
			"temperature":     gpu.Temperature,
			"power_usage":     gpu.PowerUsage,
			"active_models":   len(gpu.ActiveModels),
			"avg_latency_ms":  gpu.AvgLatency.Milliseconds(),
			"throughput":      gpu.Throughput,
			"error_rate":      gpu.ErrorRate,
		}
		gpuMetrics = append(gpuMetrics, gpuMetric)
	}
	
	metrics["gpu_details"] = gpuMetrics
	
	return metrics
}

// NewGPUScheduler creates a new GPU scheduler
func NewGPUScheduler(config *GPUOptimizationConfig) *GPUScheduler {
	scheduler := &GPUScheduler{
		queue:      make(chan *SchedulingRequest, 1000),
		workers:    10,
		algorithms: make(map[string]SchedulingAlgorithm),
	}
	
	// Register scheduling algorithms
	scheduler.algorithms["best_fit"] = &BestFitAlgorithm{}
	scheduler.algorithms["first_fit"] = &FirstFitAlgorithm{}
	scheduler.algorithms["round_robin"] = &RoundRobinAlgorithm{}
	scheduler.algorithms["least_loaded"] = &LeastLoadedAlgorithm{}
	
	// Start scheduler workers
	go scheduler.startWorkers()
	
	return scheduler
}

// SchedulingAlgorithm implementations

type BestFitAlgorithm struct{}

func (bfa *BestFitAlgorithm) SelectGPU(request *SchedulingRequest, availableGPUs []*GPUDevice) (string, error) {
	bestGPU := ""
	bestFit := uint64(^uint64(0)) // Max uint64
	
	for _, gpu := range availableGPUs {
		fit := gpu.MemoryAvailable - request.MemoryRequired
		if fit >= 0 && fit < bestFit {
			bestFit = fit
			bestGPU = gpu.ID
		}
	}
	
	if bestGPU == "" {
		return "", fmt.Errorf("no suitable GPU found for best fit")
	}
	
	return bestGPU, nil
}

func (bfa *BestFitAlgorithm) Name() string {
	return "best_fit"
}

type FirstFitAlgorithm struct{}

func (ffa *FirstFitAlgorithm) SelectGPU(request *SchedulingRequest, availableGPUs []*GPUDevice) (string, error) {
	for _, gpu := range availableGPUs {
		if gpu.MemoryAvailable >= request.MemoryRequired {
			return gpu.ID, nil
		}
	}
	
	return "", fmt.Errorf("no suitable GPU found for first fit")
}

func (ffa *FirstFitAlgorithm) Name() string {
	return "first_fit"
}

type RoundRobinAlgorithm struct {
	lastIndex int
}

func (rra *RoundRobinAlgorithm) SelectGPU(request *SchedulingRequest, availableGPUs []*GPUDevice) (string, error) {
	if len(availableGPUs) == 0 {
		return "", fmt.Errorf("no available GPUs")
	}
	
	// Find next available GPU
	for i := 0; i < len(availableGPUs); i++ {
		index := (rra.lastIndex + i + 1) % len(availableGPUs)
		gpu := availableGPUs[index]
		
		if gpu.MemoryAvailable >= request.MemoryRequired {
			rra.lastIndex = index
			return gpu.ID, nil
		}
	}
	
	return "", fmt.Errorf("no suitable GPU found for round robin")
}

func (rra *RoundRobinAlgorithm) Name() string {
	return "round_robin"
}

type LeastLoadedAlgorithm struct{}

func (lla *LeastLoadedAlgorithm) SelectGPU(request *SchedulingRequest, availableGPUs []*GPUDevice) (string, error) {
	bestGPU := ""
	lowestUtilization := float64(100)
	
	for _, gpu := range availableGPUs {
		if gpu.MemoryAvailable >= request.MemoryRequired && gpu.Utilization < lowestUtilization {
			lowestUtilization = gpu.Utilization
			bestGPU = gpu.ID
		}
	}
	
	if bestGPU == "" {
		return "", fmt.Errorf("no suitable GPU found for least loaded")
	}
	
	return bestGPU, nil
}

func (lla *LeastLoadedAlgorithm) Name() string {
	return "least_loaded"
}

// NewGPUMemoryManager creates a new GPU memory manager
func NewGPUMemoryManager(config *GPUOptimizationConfig) *GPUMemoryManager {
	return &GPUMemoryManager{
		allocator: &MemoryAllocator{
			strategy: "best_fit",
		},
		defragmenter: &MemoryDefragmenter{
			threshold: 0.8,
			interval:  5 * time.Minute,
		},
		reservation: &MemoryReservation{
			reservations: make(map[string]*Reservation),
		},
	}
}

// NewGPUPerformanceMonitor creates a new GPU performance monitor
func NewGPUPerformanceMonitor() *GPUPerformanceMonitor {
	return &GPUPerformanceMonitor{
		metrics: make(map[string]*GPUMetrics),
	}
}

// ModelUsageInfo represents model usage information
type ModelUsageInfo struct {
	ModelID        string
	ModelName      string
	MemoryRequired uint64
	RequestCount   int64
	LastAccessed   time.Time
}

func maxTime(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}