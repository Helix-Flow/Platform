#!/usr/bin/env python3

import os
import sys
import json
import time
import logging
from typing import Dict, Any, List, Optional
from datetime import datetime

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class MonitoringService:
    """Simple Monitoring Service implementation for testing purposes."""
    
    def __init__(self):
        self.port = int(os.getenv('MONITORING_PORT', 8083))
        self.health_status = "healthy"
        self.metrics = {
            "requests_total": 0,
            "requests_failed": 0,
            "average_response_time": 0.0,
            "active_connections": 0
        }
        self.alerts = []
        
    def health_check(self) -> Dict[str, Any]:
        """Health check endpoint."""
        return {
            "status": self.health_status,
            "timestamp": int(time.time()),
            "service": "monitoring",
            "version": "1.0.0"
        }
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get current metrics."""
        return {
            "metrics": self.metrics,
            "timestamp": int(time.time()),
            "collection_time": datetime.now().isoformat()
        }
    
    def record_request(self, response_time: float, success: bool = True) -> None:
        """Record a request metric."""
        self.metrics["requests_total"] += 1
        self.metrics["active_connections"] += 1
        
        if not success:
            self.metrics["requests_failed"] += 1
        
        # Update average response time
        total_time = self.metrics["average_response_time"] * (self.metrics["requests_total"] - 1)
        self.metrics["average_response_time"] = (total_time + response_time) / self.metrics["requests_total"]
        
        logger.info(f"Request recorded: response_time={response_time:.3f}s, success={success}")
    
    def record_connection_closed(self) -> None:
        """Record that a connection was closed."""
        if self.metrics["active_connections"] > 0:
            self.metrics["active_connections"] -= 1
        
        logger.info("Connection closed")
    
    def create_alert(self, alert_type: str, message: str, severity: str = "info") -> Dict[str, Any]:
        """Create an alert."""
        alert = {
            "id": f"alert-{int(time.time())}",
            "type": alert_type,
            "message": message,
            "severity": severity,
            "timestamp": int(time.time()),
            "created_at": datetime.now().isoformat()
        }
        
        self.alerts.append(alert)
        logger.warning(f"Alert created: {alert_type} - {message}")
        
        return alert
    
    def get_alerts(self, severity: Optional[str] = None) -> Dict[str, Any]:
        """Get alerts, optionally filtered by severity."""
        if severity:
            filtered_alerts = [alert for alert in self.alerts if alert["severity"] == severity]
        else:
            filtered_alerts = self.alerts
        
        return {
            "alerts": filtered_alerts,
            "total_count": len(filtered_alerts)
        }
    
    def clear_alerts(self) -> Dict[str, Any]:
        """Clear all alerts."""
        count = len(self.alerts)
        self.alerts.clear()
        logger.info(f"Cleared {count} alerts")
        
        return {
            "message": f"Cleared {count} alerts",
            "cleared_count": count
        }
    
    def generate_report(self) -> Dict[str, Any]:
        """Generate a comprehensive monitoring report."""
        success_rate = 0.0
        if self.metrics["requests_total"] > 0:
            success_rate = ((self.metrics["requests_total"] - self.metrics["requests_failed"]) / 
                           self.metrics["requests_total"] * 100)
        
        return {
            "report": {
                "summary": {
                    "total_requests": self.metrics["requests_total"],
                    "failed_requests": self.metrics["requests_failed"],
                    "success_rate": round(success_rate, 2),
                    "average_response_time": round(self.metrics["average_response_time"], 3),
                    "active_connections": self.metrics["active_connections"]
                },
                "alerts": {
                    "total": len(self.alerts),
                    "by_severity": {
                        "critical": len([a for a in self.alerts if a["severity"] == "critical"]),
                        "warning": len([a for a in self.alerts if a["severity"] == "warning"]),
                        "info": len([a for a in self.alerts if a["severity"] == "info"])
                    }
                },
                "generated_at": datetime.now().isoformat()
            }
        }

def main():
    """Main function for testing."""
    monitoring = MonitoringService()
    
    # Test health check
    health = monitoring.health_check()
    print(f"Health Check: {json.dumps(health, indent=2)}")
    
    # Test recording some metrics
    monitoring.record_request(0.123, success=True)
    monitoring.record_request(0.234, success=True)
    monitoring.record_request(0.345, success=False)
    
    # Test metrics retrieval
    metrics = monitoring.get_metrics()
    print(f"Metrics: {json.dumps(metrics, indent=2)}")
    
    # Test alert creation
    alert = monitoring.create_alert("high_cpu", "CPU usage above 90%", "warning")
    print(f"Alert: {json.dumps(alert, indent=2)}")
    
    # Test report generation
    report = monitoring.generate_report()
    print(f"Report: {json.dumps(report, indent=2)}")

if __name__ == "__main__":
    main()
