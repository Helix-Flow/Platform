package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

type PredictiveScaler struct {
	redisClient     *redis.Client
	scalingHistory  []ScalingEvent
	predictionWindow time.Duration
	minReplicas     int
	maxReplicas     int
}

type ScalingEvent struct {
	Timestamp   time.Time
	CPUUsage    float64
	MemoryUsage float64
	RequestRate float64
	Replicas    int
}

type ScalingPrediction struct {
	Timestamp      time.Time
	PredictedLoad  float64
	RecommendedReplicas int
	Confidence     float64
}

func NewPredictiveScaler(redisClient *redis.Client) *PredictiveScaler {
	return &PredictiveScaler{
		redisClient:     redisClient,
		scalingHistory:  make([]ScalingEvent, 0, 1000),
		predictionWindow: 24 * time.Hour,
		minReplicas:     3,
		maxReplicas:     20,
	}
}

func (ps *PredictiveScaler) RecordMetrics(cpu, memory, requests float64, replicas int) {
	event := ScalingEvent{
		Timestamp:   time.Now(),
		CPUUsage:    cpu,
		MemoryUsage: memory,
		RequestRate: requests,
		Replicas:    replicas,
	}

	ps.scalingHistory = append(ps.scalingHistory, event)

	// Keep only recent history
	if len(ps.scalingHistory) > 1000 {
		ps.scalingHistory = ps.scalingHistory[1:]
	}

	// Store in Redis for persistence
	ps.storeEvent(event)
}

func (ps *PredictiveScaler) PredictScaling() *ScalingPrediction {
	if len(ps.scalingHistory) < 10 {
		return &ScalingPrediction{
			Timestamp:         time.Now(),
			PredictedLoad:     0.5,
			RecommendedReplicas: ps.minReplicas,
			Confidence:        0.5,
		}
	}

	// Simple linear regression for prediction
	loadPrediction := ps.predictLoad()
	replicas := ps.calculateOptimalReplicas(loadPrediction)

	return &ScalingPrediction{
		Timestamp:         time.Now(),
		PredictedLoad:     loadPrediction,
		RecommendedReplicas: replicas,
		Confidence:        ps.calculateConfidence(),
	}
}

func (ps *PredictiveScaler) predictLoad() float64 {
	if len(ps.scalingHistory) < 2 {
		return 0.5
	}

	// Calculate trend using recent data (last hour)
	recentEvents := ps.getRecentEvents(time.Hour)

	if len(recentEvents) < 2 {
		return recentEvents[len(recentEvents)-1].CPUUsage
	}

	// Simple linear regression
	n := float64(len(recentEvents))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, event := range recentEvents {
		x := float64(i)
		y := event.CPUUsage
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Predict next value
	nextX := float64(len(recentEvents))
	prediction := slope*nextX + intercept

	// Clamp to reasonable range
	if prediction < 0 {
		prediction = 0
	}
	if prediction > 1 {
		prediction = 1
	}

	return prediction
}

func (ps *PredictiveScaler) calculateOptimalReplicas(load float64) int {
	// Calculate replicas based on load
	// Target: 70% utilization per replica
	targetUtilization := 0.7
	replicas := int(math.Ceil(load / targetUtilization))

	// Clamp to min/max
	if replicas < ps.minReplicas {
		replicas = ps.minReplicas
	}
	if replicas > ps.maxReplicas {
		replicas = ps.maxReplicas
	}

	return replicas
}

func (ps *PredictiveScaler) calculateConfidence() float64 {
	if len(ps.scalingHistory) < 10 {
		return 0.5
	}

	// Calculate coefficient of determination (R²)
	recentEvents := ps.getRecentEvents(2 * time.Hour)
	if len(recentEvents) < 3 {
		return 0.5
	}

	// Simplified confidence calculation
	variance := ps.calculateVariance(recentEvents)
	if variance == 0 {
		return 1.0
	}

	// Higher confidence with lower variance
	confidence := 1.0 - math.Min(variance/0.1, 1.0)
	return math.Max(confidence, 0.1)
}

func (ps *PredictiveScaler) calculateVariance(events []ScalingEvent) float64 {
	if len(events) < 2 {
		return 0
	}

	mean := 0.0
	for _, event := range events {
		mean += event.CPUUsage
	}
	mean /= float64(len(events))

	variance := 0.0
	for _, event := range events {
		diff := event.CPUUsage - mean
		variance += diff * diff
	}
	variance /= float64(len(events) - 1)

	return variance
}

func (ps *PredictiveScaler) getRecentEvents(duration time.Duration) []ScalingEvent {
	cutoff := time.Now().Add(-duration)
	events := make([]ScalingEvent, 0)

	for _, event := range ps.scalingHistory {
		if event.Timestamp.After(cutoff) {
			events = append(events, event)
		}
	}

	return events
}

func (ps *PredictiveScaler) storeEvent(event ScalingEvent) {
	key := fmt.Sprintf("scaling_event:%d", event.Timestamp.Unix())
	data := fmt.Sprintf("%d,%.3f,%.3f,%.3f,%d",
		event.Timestamp.Unix(),
		event.CPUUsage,
		event.MemoryUsage,
		event.RequestRate,
		event.Replicas)

	err := ps.redisClient.Set(context.Background(), key, data, 7*24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to store scaling event: %v", err)
	}
}

func (ps *PredictiveScaler) GetScalingStats() map[string]interface{} {
	recentEvents := ps.getRecentEvents(time.Hour)

	return map[string]interface{}{
		"total_events":     len(ps.scalingHistory),
		"recent_events":    len(recentEvents),
		"avg_cpu_last_hour": ps.calculateAverageCPU(recentEvents),
		"prediction_window": ps.predictionWindow.String(),
		"min_replicas":     ps.minReplicas,
		"max_replicas":     ps.maxReplicas,
	}
}

func (ps *PredictiveScaler) calculateAverageCPU(events []ScalingEvent) float64 {
	if len(events) == 0 {
		return 0.0
	}

	total := 0.0
	for _, event := range events {
		total += event.CPUUsage
	}
	return total / float64(len(events))
}
