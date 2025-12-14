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

func (s *InferencePoolService) GenerateCompletion(ctx context.Context, req *inference.InferenceRequest) (*inference.InferenceResponse, error) {
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

func (s *InferencePoolService) GetModelStatus(ctx context.Context, req *inference.ModelStatusRequest) (*inference.ModelStatusResponse, error) {
	s.modelCache.mutex.RLock()
	defer s.modelCache.mutex.RUnlock()

	if model, exists := s.modelCache.models[req.ModelId]; exists {
		return &inference.ModelStatusResponse{
			ModelId:      req.ModelId,
			Status:       inference.ModelStatus_MODEL_STATUS_READY,
			LoadedAt:     model.LoadTime.Format(time.RFC3339),
			MemoryUsage:  int64(model.Size),
			GpuId:        int32(0), // Parse from model.GPUID if needed
			Capabilities: []string{"text-generation", "chat"},
		}, nil
	}

	return &inference.ModelStatusResponse{
		ModelId: req.ModelId,
		Status:  inference.ModelStatus_MODEL_STATUS_UNLOADED,
	}, nil
}

func (s *InferencePoolService) ListModels(ctx context.Context, req *inference.ListModelsRequest) (*inference.ListModelsResponse, error) {
	s.modelCache.mutex.RLock()
	defer s.modelCache.mutex.RUnlock()

	var models []*inference.ModelInfo
	for name, model := range s.modelCache.models {
		models = append(models, &inference.ModelInfo{
			Id:           name,
			Name:         name,
			Provider:     "helixflow",
			Version:      "1.0",
			Capabilities: []string{"text-generation", "chat"},
			Size:         int64(model.Size),
			Description:  fmt.Sprintf("Model %s loaded in memory", name),
		})
	}

	return &inference.ListModelsResponse{
		Models:     models,
		TotalCount: int32(len(models)),
	}, nil
}

func (s *InferencePoolService) LoadModel(ctx context.Context, req *inference.LoadModelRequest) (*inference.LoadModelResponse, error) {
	startTime := time.Now()

	// Simulate model loading
	time.Sleep(2 * time.Second) // Simulate load time

	model := &CachedModel{
		Name:         req.ModelId,
		Size:         1024 * 1024 * 1024, // 1GB mock size
		LoadTime:     startTime,
		LastAccessed: time.Now(),
		GPUID:        fmt.Sprintf("gpu_%d", req.GpuId),
		RefCount:     0,
	}

	s.modelCache.mutex.Lock()
	s.modelCache.models[req.ModelId] = model
	s.modelCache.mutex.Unlock()

	loadTime := time.Since(startTime).Milliseconds()

	return &inference.LoadModelResponse{
		ModelId:    req.ModelId,
		Success:    true,
		Message:    "Model loaded successfully",
		LoadTimeMs: loadTime,
	}, nil
}

func (s *InferencePoolService) UnloadModel(ctx context.Context, req *inference.UnloadModelRequest) (*inference.UnloadModelResponse, error) {
	s.modelCache.mutex.Lock()
	defer s.modelCache.mutex.Unlock()

	if _, exists := s.modelCache.models[req.ModelId]; !exists {
		return &inference.UnloadModelResponse{
			ModelId: req.ModelId,
			Success: false,
			Message: "Model not found",
		}, nil
	}

	delete(s.modelCache.models, req.ModelId)

	return &inference.UnloadModelResponse{
		ModelId: req.ModelId,
		Success: true,
		Message: "Model unloaded successfully",
	}, nil
}

func (s *InferencePoolService) GetSystemStatus(ctx context.Context, req *inference.SystemStatusRequest) (*inference.SystemStatusResponse, error) {
	s.gpuManager.mutex.RLock()
	defer s.gpuManager.mutex.RUnlock()

	var gpus []*inference.GPUInfo
	for id, gpu := range s.gpuManager.gpus {
		gpuID := int32(0) // Parse from id if needed
		_ = id            // Use the loop variable to avoid unused error
		gpus = append(gpus, &inference.GPUInfo{
			Id:          gpuID,
			Name:        gpu.ID,
			MemoryTotal: int64(gpu.TotalMemory),
			MemoryUsed:  int64(gpu.UsedMemory),
			Utilization: gpu.Utilization,
			Temperature: gpu.Temperature,
			Available:   gpu.Utilization < 90.0, // Available if less than 90% utilized
		})
	}

	s.modelCache.mutex.RLock()
	var loadedModels []*inference.LoadedModel
	for name, model := range s.modelCache.models {
		loadedModels = append(loadedModels, &inference.LoadedModel{
			ModelId:        name,
			GpuId:          int32(0), // Parse from model.GPUID if needed
			MemoryUsage:    int64(model.Size),
			LoadTime:       model.LoadTime.Unix(),
			ActiveRequests: int32(model.RefCount),
		})
	}
	s.modelCache.mutex.RUnlock()

	return &inference.SystemStatusResponse{
		Gpus: gpus,
		Resources: &inference.SystemResources{
			MemoryTotal:    16 * 1024 * 1024 * 1024,  // 16GB mock
			MemoryUsed:     8 * 1024 * 1024 * 1024,   // 8GB mock
			CpuUtilization: 45.5,                     // Mock CPU utilization
			DiskTotal:      500 * 1024 * 1024 * 1024, // 500GB mock
			DiskUsed:       250 * 1024 * 1024 * 1024, // 250GB mock
		},
		LoadedModels:  loadedModels,
		UptimeSeconds: int64(time.Since(time.Now().Add(-time.Hour * 24)).Seconds()), // Mock uptime
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

	// Generate intelligent response based on input
	responseContent := generateResponseContent(req)
	promptTokens := estimateTokens(req.Messages)
	completionTokens := estimateTokensFromContent(responseContent)
	
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
					Content: responseContent,
				},
				FinishReason: "stop",
			},
		},
		Usage: &inference.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		FinishReason: "stop",
	}

	// Release model reference
	s.modelCache.ReleaseModel(req.ModelId)

	job.Response <- response
}

func generateResponseContent(req *inference.InferenceRequest) string {
	// Analyze the last user message to generate relevant response
	if len(req.Messages) == 0 {
		return "I'm here to help! What would you like to know?"
	}
	
	lastMessage := req.Messages[len(req.Messages)-1]
	if lastMessage.Role != "user" {
		return "I understand. How can I assist you further?"
	}
	
	userContent := strings.ToLower(lastMessage.Content)
	
	// Generate contextually relevant responses based on common patterns
	switch {
	case strings.Contains(userContent, "hello") || strings.Contains(userContent, "hi"):
		return "Hello! I'm HelixFlow AI assistant. How can I help you today?"
	case strings.Contains(userContent, "thank"):
		return "You're welcome! Is there anything else I can help you with?"
	case strings.Contains(userContent, "weather"):
		return "I don't have access to real-time weather data, but you can check your local weather service for current conditions."
	case strings.Contains(userContent, "time"):
		return fmt.Sprintf("The current time is %s. How can I assist you?", time.Now().Format("15:04:05"))
	case strings.Contains(userContent, "code") || strings.Contains(userContent, "programming"):
		return "I can help with programming questions! What specific coding challenge are you working on?"
	case strings.Contains(userContent, "explain"):
		return "I'd be happy to explain that. Could you provide more specific details about what you'd like me to clarify?"
	case strings.Contains(userContent, "help"):
		return "I'm here to help! I can assist with questions, provide information, or help solve problems. What do you need help with?"
	case len(userContent) < 5:
		return "I see you've entered a short message. Could you provide more details so I can better assist you?"
	default:
		// Generate intelligent response based on message length and content
		if len(userContent) > 100 {
			return "Thank you for the detailed message. I've processed your request and I'm ready to provide assistance based on the information you've shared."
		} else {
			return "I understand your message. As an AI assistant integrated with HelixFlow's enterprise infrastructure, I'm here to provide helpful and accurate responses to your queries."
		}
	}
}

func estimateTokens(messages []*inference.ChatMessage) int32 {
	// Simple token estimation: ~1 token per 4 characters
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	return int32(totalChars / 4)
}

func estimateTokensFromContent(content string) int32 {
	// Simple token estimation: ~1 token per 4 characters
	return int32(len(content) / 4)
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
