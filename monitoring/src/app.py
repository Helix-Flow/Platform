"""
HelixFlow Monitoring Service

Prometheus metrics collection and alerting
"""

from fastapi import FastAPI
from prometheus_client import (
    Counter,
    Histogram,
    Gauge,
    generate_latest,
    CONTENT_TYPE_LATEST,
)
from starlette.responses import Response
import logging
from datetime import datetime
import redis
import os
import psutil
import GPUtil

app = FastAPI(title="HelixFlow Monitoring", version="1.0.0")

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Redis for metrics storage
redis_client = redis.Redis(
    host=os.getenv("REDIS_HOST", "localhost"),
    port=int(os.getenv("REDIS_PORT", 6379)),
    password=os.getenv("REDIS_PASSWORD"),
    decode_responses=True,
)

# Prometheus metrics
REQUEST_COUNT = Counter(
    "helixflow_requests_total",
    "Total number of requests",
    ["service", "endpoint", "method", "status"],
)

REQUEST_LATENCY = Histogram(
    "helixflow_request_duration_seconds",
    "Request duration in seconds",
    ["service", "endpoint"],
)

GPU_MEMORY_USAGE = Gauge(
    "helixflow_gpu_memory_usage_bytes", "GPU memory usage in bytes", ["gpu_id"]
)

GPU_UTILIZATION = Gauge(
    "helixflow_gpu_utilization_percent", "GPU utilization percentage", ["gpu_id"]
)

ACTIVE_CONNECTIONS = Gauge(
    "helixflow_active_connections", "Number of active connections", ["service"]
)

QUEUE_SIZE = Gauge("helixflow_queue_size", "Current queue size", ["queue_name"])

MODEL_INFERENCE_COUNT = Counter(
    "helixflow_model_inferences_total",
    "Total number of model inferences",
    ["model_name", "gpu_id"],
)

ERROR_COUNT = Counter(
    "helixflow_errors_total", "Total number of errors", ["service", "error_type"]
)


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "service": "monitoring",
    }


@app.get("/metrics")
async def metrics():
    """Prometheus metrics endpoint"""
    # Update system metrics
    update_system_metrics()

    # Update GPU metrics
    update_gpu_metrics()

    # Update queue metrics
    update_queue_metrics()

    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)


@app.post("/metrics/request")
async def record_request(
    service: str, endpoint: str, method: str, status: int, duration: float
):
    """Record request metrics"""
    REQUEST_COUNT.labels(
        service=service, endpoint=endpoint, method=method, status=status
    ).inc()

    REQUEST_LATENCY.labels(service=service, endpoint=endpoint).observe(duration)


@app.post("/metrics/inference")
async def record_inference(model_name: str, gpu_id: str):
    """Record model inference metrics"""
    MODEL_INFERENCE_COUNT.labels(model_name=model_name, gpu_id=gpu_id).inc()


@app.post("/metrics/error")
async def record_error(service: str, error_type: str):
    """Record error metrics"""
    ERROR_COUNT.labels(service=service, error_type=error_type).inc()


def update_system_metrics():
    """Update system-level metrics"""
    # CPU usage
    cpu_percent = psutil.cpu_percent()
    # Memory usage
    memory = psutil.virtual_memory()
    # Disk usage
    disk = psutil.disk_usage("/")

    # Update gauges (would need to define these metrics)
    # For now, just log
    logger.info(
        f"System metrics - CPU: {cpu_percent}%, Memory: {memory.percent}%, Disk: {disk.percent}%"
    )


def update_gpu_metrics():
    """Update GPU metrics"""
    try:
        gpus = GPUtil.getGPUs()
        for i, gpu in enumerate(gpus):
            GPU_MEMORY_USAGE.labels(gpu_id=f"gpu_{i}").set(
                gpu.memoryUsed * 1024 * 1024
            )  # Convert to bytes
            GPU_UTILIZATION.labels(gpu_id=f"gpu_{i}").set(gpu.load * 100)
    except Exception as e:
        logger.error(f"Failed to get GPU metrics: {e}")


def update_queue_metrics():
    """Update queue size metrics"""
    try:
        # Check Redis queues
        inference_queue_size = redis_client.llen("inference_queue")
        QUEUE_SIZE.labels(queue_name="inference").set(inference_queue_size)

        # Other queues could be added here
    except Exception as e:
        logger.error(f"Failed to get queue metrics: {e}")


@app.get("/alerts")
async def get_alerts():
    """Get current alerts"""
    # In a real implementation, this would query Prometheus Alertmanager
    alerts = [
        {
            "alertname": "HighGPUUtilization",
            "severity": "warning",
            "description": "GPU utilization above 90%",
            "state": "firing",
        }
    ]
    return {"alerts": alerts}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8002)
