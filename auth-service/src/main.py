#!/usr/bin/env python3

import os
import sys
import json
import time
import logging
import hashlib
from typing import Dict, Any, Optional

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class AuthService:
    """Simple Auth Service implementation for testing purposes."""
    
    def __init__(self):
        self.port = int(os.getenv('AUTH_SERVICE_PORT', 8081))
        self.health_status = "healthy"
        self.users = {}  # Simple in-memory user store for testing
        
    def health_check(self) -> Dict[str, Any]:
        """Health check endpoint."""
        return {
            "status": self.health_status,
            "timestamp": int(time.time()),
            "service": "auth-service",
            "version": "1.0.0"
        }
    
    def register_user(self, username: str, email: str, password: str) -> Dict[str, Any]:
        """Register a new user."""
        try:
            if username in self.users:
                return {"error": "Username already exists"}
            
            # Simple password hashing (not for production)
            password_hash = hashlib.sha256(password.encode()).hexdigest()
            
            self.users[username] = {
                "username": username,
                "email": email,
                "password_hash": password_hash,
                "created_at": int(time.time())
            }
            
            logger.info(f"User registered: {username}")
            return {
                "message": "User registered successfully",
                "username": username,
                "user_id": hashlib.md5(username.encode()).hexdigest()
            }
        except Exception as e:
            logger.error(f"Error registering user: {e}")
            return {"error": str(e)}
    
    def login_user(self, username: str, password: str) -> Dict[str, Any]:
        """Login a user."""
        try:
            if username not in self.users:
                return {"error": "Invalid credentials"}
            
            user = self.users[username]
            password_hash = hashlib.sha256(password.encode()).hexdigest()
            
            if user["password_hash"] != password_hash:
                return {"error": "Invalid credentials"}
            
            # Generate simple token (not for production)
            token = hashlib.sha256(f"{username}{time.time()}".encode()).hexdigest()
            
            logger.info(f"User logged in: {username}")
            return {
                "access_token": token,
                "token_type": "bearer",
                "username": username
            }
        except Exception as e:
            logger.error(f"Error logging in user: {e}")
            return {"error": str(e)}

def main():
    """Main function for testing."""
    auth = AuthService()
    
    # Test health check
    health = auth.health_check()
    print(f"Health Check: {json.dumps(health, indent=2)}")
    
    # Test user registration
    reg_result = auth.register_user("testuser", "test@example.com", "password")
    print(f"Registration: {json.dumps(reg_result, indent=2)}")
    
    # Test user login
    login_result = auth.login_user("testuser", "password")
    print(f"Login: {json.dumps(login_result, indent=2)}")

if __name__ == "__main__":
    main()
