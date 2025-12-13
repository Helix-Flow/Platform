package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"helixflow/inference"
)

// GPUManager handles GPU resource allocation and monitoring
type GPUManager struct {
	gpus  map[string]*GPUInfo
	mutex sync.RWMutex
}

type GPUInfo struct {
	ID           string    `json:"id"`
	TotalMemory  uint64    `json:"total_memory"` // in bytes
	UsedMemory   uint64    `json:"used_memory"`  // in bytes
	Utilization  float64   `json:"utilization"`  // percentage
	Temperature  float64   `json:"temperature"`  // celsius
	ActiveModels []string  `json:"active_models"`
	LastUpdated  time.Time `json:"last_updated"`
}

type ModelCache struct {
	models map[string]*CachedModel
	mutex  sync.RWMutex
}

type CachedModel struct {
	Name         string    `json:"name"`
	Size         uint64    `json:"size"` // in bytes
	LoadTime     time.Time `json:"load_time"`
	LastAccessed time.Time `json:"last_accessed"`
	GPUID        string    `json:"gpu_id"`
	RefCount     int       `json:"ref_count"`
}

// InferencePoolService implements the gRPC service
type InferencePoolService struct {
	inference.UnimplementedInferenceServiceServer
	gpuManager *GPUManager
	modelCache *ModelCache
	jobQueue   chan *InferenceJob
	workers    int
}

type InferenceJob struct {
	ID       string
	Request  *inference.InferenceRequest
	Response chan *inference.InferenceResponse
	Error    chan error
}

func NewGPUManager() *GPUManager {
	gm := &GPUManager{
		gpus: make(map[string]*GPUInfo),
	}

	// Initialize with mock GPUs (in real implementation, detect actual GPUs)
	gm.initializeGPUs()

	return gm
}

func (gm *GPUManager) initializeGPUs() {
	gpus := []string{"gpu_0", "gpu_1", "gpu_2", "gpu_3"}
	for _, gpuID := range gpus {
		gm.gpus[gpuID] = &GPUInfo{
			ID:           gpuID,
			TotalMemory:  24 * 1024 * 1024 * 1024, // 24GB
			UsedMemory:   0,
			Utilization:  0.0,
			Temperature:  35.0,
			ActiveModels: []string{},
			LastUpdated:  time.Now(),
		}
	}
}

func (gm *GPUManager) AllocateGPU(modelName string, requiredMemory uint64) (string, error) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	// Find best GPU for the model
	var bestGPU string
	var bestScore float64 = -1

	for gpuID, gpu := range gm.gpus {
		availableMemory := gpu.TotalMemory - gpu.UsedMemory
		if availableMemory >= requiredMemory {
			// Score based on available memory and current utilization
			score := float64(availableMemory) / float64(gpu.TotalMemory) * (1 - gpu.Utilization/100)
			if score > bestScore {
				bestScore = score
				bestGPU = gpuID
			}
		}
	}

	if bestGPU == "" {
		return "", fmt.Errorf("no GPU available for model %s requiring %d bytes", modelName, requiredMemory)
	}

	// Allocate memory
	gm.gpus[bestGPU].UsedMemory += requiredMemory
	gm.gpus[bestGPU].ActiveModels = append(gm.gpus[bestGPU].ActiveModels, modelName)
	gm.gpus[bestGPU].LastUpdated = time.Now()

	return bestGPU, nil
}

func (gm *GPUManager) FreeGPU(gpuID string, modelName string, memory uint64) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	if gpu, exists := gm.gpus[gpuID]; exists {
		if gpu.UsedMemory >= memory {
			gpu.UsedMemory -= memory
		}

		// Remove model from active list
		for i, model := range gpu.ActiveModels {
			if model == modelName {
				gpu.ActiveModels = append(gpu.ActiveModels[:i], gpu.ActiveModels[i+1:]...)
				break
			}
		}

		gpu.LastUpdated = time.Now()
	}
}

func (gm *GPUManager) GetGPUStatus() map[string]*GPUInfo {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	result := make(map[string]*GPUInfo)
	for id, gpu := range gm.gpus {
		gpuCopy := *gpu
		result[id] = &gpuCopy
	}
	return result
}

func NewModelCache() *ModelCache {
	return &ModelCache{
		models: make(map[string]*CachedModel),
	}
}

func (mc *ModelCache) GetModel(modelName string) *CachedModel {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if model, exists := mc.models[modelName]; exists {
		model.LastAccessed = time.Now()
		model.RefCount++
		return model
	}
	return nil
}

func (mc *ModelCache) CacheModel(modelName string, size uint64, gpuID string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.models[modelName] = &CachedModel{
		Name:         modelName,
		Size:         size,
		LoadTime:     time.Now(),
		LastAccessed: time.Now(),
		GPUID:        gpuID,
		RefCount:     1,
	}
}

func (mc *ModelCache) ReleaseModel(modelName string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if model, exists := mc.models[modelName]; exists {
		model.RefCount--
		if model.RefCount <= 0 {
			delete(mc.models, modelName)
		}
	}
}

func (mc *ModelCache) EvictLRU() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	var oldestModel string
	var oldestTime time.Time

	for name, model := range mc.models {
		if oldestModel == "" || model.LastAccessed.Before(oldestTime) {
			oldestModel = name
			oldestTime = model.LastAccessed
		}
	}

	if oldestModel != "" {
		delete(mc.models, oldestModel)
	}
}

func NewInferencePoolService() *InferencePoolService {
	return &InferencePoolService{
		gpuManager: NewGPUManager(),
		modelCache: NewModelCache(),
		jobQueue:   make(chan *InferenceJob, 1000),
		workers:    10,
	}
}

func (s *InferencePoolService) SubmitInference(ctx context.Context, req *inference.InferenceRequest) (*inference.InferenceResponse, error) {
	job := &InferenceJob{
		ID:       fmt.Sprintf("job_%d", time.Now().UnixNano()),
		Request:  req,
		Response: make(chan *inference.InferenceResponse, 1),
		Error:    make(chan error, 1),
	}

	// Add to queue
	select {
	case s.jobQueue <- job:
		// Job queued successfully
	default:
		return nil, fmt.Errorf("job queue full")
	}

	// Wait for response with timeout
	select {
	case response := <-job.Response:
		return response, nil
	case err := <-job.Error:
		return nil, err
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("inference timeout")
	}
}

func (s *InferencePoolService) GetSystemStatus(ctx context.Context, req *inference.SystemStatusRequest) (*inference.SystemStatusResponse, error) {
	// In a real implementation, check job status from storage
	return &inference.SystemStatusResponse{
		// JobId:  req.JobId,
		// Status: "completed",
		// Result: "Mock inference result",
	}, nil
}

func (s *InferencePoolService) StreamInference(req *inference.InferenceRequest, stream inference.InferenceService_StreamInferenceServer) error {
	// Simulate streaming response
	content := fmt.Sprintf("Streaming response for model %s", req.ModelId)

	words := []string{}
	for _, word := range strings.Fields(content) {
		words = append(words, word)
	}

	for _, word := range words {
		chunk := &inference.InferenceChunk{
			Id:      fmt.Sprintf("chunk_%d", time.Now().UnixNano()),
			Object:  "inference.chunk",
			Created: time.Now().Unix(),
			Model:   req.ModelId,
			Choices: []*inference.Choice{
				{
					Index: 0,
					Delta: &inference.Delta{
						Content: word + " ",
					},
				},
			},
		}

		if err := stream.Send(chunk); err != nil {
			return err
		}

		time.Sleep(100 * time.Millisecond) // Simulate processing delay
	}

	return nil
}

func (s *InferencePoolService) startWorkers() {
	for i := 0; i < s.workers; i++ {
		go s.worker()
	}
}

func (s *InferencePoolService) worker() {
	for job := range s.jobQueue {
		s.processJob(job)
	}
}

func (s *InferencePoolService) processJob(job *InferenceJob) {
	defer close(job.Response)
	defer close(job.Error)

	req := job.Request

	// Check model cache
	cachedModel := s.modelCache.GetModel(req.ModelId)
	if cachedModel == nil {
		// Model not cached, allocate GPU and load model
		modelSize := uint64(4 * 1024 * 1024 * 1024) // 4GB mock size
		gpuID, err := s.gpuManager.AllocateGPU(req.ModelId, modelSize)
		if err != nil {
			job.Error <- err
			return
		}

		// Cache the model
		s.modelCache.CacheModel(req.ModelId, modelSize, gpuID)

		// Simulate model loading time
		time.Sleep(2 * time.Second)
	} else {
		// Model already cached
		_ = cachedModel.GPUID // Use the GPUID
	}

	// Simulate inference processing
	time.Sleep(500 * time.Millisecond)

	// Generate mock response
	response := &inference.InferenceResponse{
		Id:      job.ID,
		Object:  "inference.response",
		Created: time.Now().Unix(),
		Model:   req.ModelId,
		Choices: []*inference.Choice{
			{
				Index: 0,
				Message: &inference.ChatMessage{
					Role:    "assistant",
					Content: "Mock response",
				},
				FinishReason: "stop",
			},
		},
		Usage: &inference.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	// Release model reference
	s.modelCache.ReleaseModel(req.ModelId)

	job.Response <- response
}

func (s *InferencePoolService) GetGPUStatus() map[string]*GPUInfo {
	return s.gpuManager.GetGPUStatus()
}

func main() {
	service := NewInferencePoolService()
	service.startWorkers()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	server := grpc.NewServer()
	inference.RegisterInferenceServiceServer(server, service)
	reflection.Register(server)

	log.Printf("Inference Pool gRPC server starting on :50051")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
