"""
API Key Management Service

Handles API key generation, validation, and rate limiting for HelixFlow.
"""

import os
import secrets
import hashlib
import bcrypt
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
import redis
import logging
from dataclasses import dataclass

logger = logging.getLogger(__name__)

# Redis for API key storage and rate limiting
redis_client = redis.Redis(
    host=os.getenv("REDIS_HOST", "localhost"),
    port=int(os.getenv("REDIS_PORT", 6379)),
    password=os.getenv("REDIS_PASSWORD"),
    decode_responses=True,
)


@dataclass
class APIKey:
    key_id: str
    user_id: str
    key_hash: str
    name: str
    created_at: datetime
    last_used: Optional[datetime]
    expires_at: Optional[datetime]
    rate_limit: int  # requests per minute
    status: str  # active, suspended, expired


class APIKeyService:
    """Service for managing API keys."""

    def __init__(self):
        self.key_length = 32
        self.hash_algorithm = "sha256"

    def generate_api_key(
        self,
        user_id: str,
        name: str = "Default Key",
        rate_limit: int = 60,
        expires_days: int = 365,
    ) -> tuple[str, str]:
        """Generate a new API key for user.

        Returns (key_id, api_key)
        """
        # Generate random key
        api_key = secrets.token_urlsafe(self.key_length)

        # Generate key ID
        key_id = secrets.token_hex(16)

        # Hash the key for storage
        key_hash = self._hash_key(api_key)

        # Create key record
        key_record = APIKey(
            key_id=key_id,
            user_id=user_id,
            key_hash=key_hash,
            name=name,
            created_at=datetime.utcnow(),
            last_used=None,
            expires_at=datetime.utcnow() + timedelta(days=expires_days),
            rate_limit=rate_limit,
            status="active",
        )

        # Store in Redis
        redis_client.setex(
            f"api_key:{key_id}",
            expires_days * 24 * 60 * 60,  # expiration in seconds
            str(key_record.__dict__),
        )

        # Index by user
        redis_client.sadd(f"user_keys:{user_id}", key_id)

        logger.info(f"Generated API key {key_id} for user {user_id}")
        return key_id, api_key

    def validate_api_key(self, api_key: str) -> Optional[APIKey]:
        """Validate API key and return key record if valid."""
        try:
            # Hash the provided key
            key_hash = self._hash_key(api_key)

            # Find key by hash (this is inefficient in real implementation)
            # In production, use a database with proper indexing
            keys = redis_client.keys("api_key:*")
            for key_key in keys:
                key_data = redis_client.get(key_key)
                if key_data:
                    key_dict = eval(key_data)  # Use JSON in production
                    if key_dict["key_hash"] == key_hash:
                        key_record = APIKey(**key_dict)

                        # Check expiration
                        if (
                            key_record.expires_at
                            and datetime.utcnow() > key_record.expires_at
                        ):
                            key_record.status = "expired"
                            self._update_key_record(key_record)
                            return None

                        # Check status
                        if key_record.status != "active":
                            return None

                        # Update last used
                        key_record.last_used = datetime.utcnow()
                        self._update_key_record(key_record)

                        return key_record

            return None

        except Exception as e:
            logger.error(f"Error validating API key: {e}")
            return None

    def check_rate_limit(self, key_id: str, rate_limit: int) -> bool:
        """Check if request is within rate limit."""
        try:
            # Use sliding window rate limiting
            current_minute = datetime.utcnow().replace(second=0, microsecond=0)
            window_key = f"rate_limit:{key_id}:{current_minute.isoformat()}"

            # Increment counter
            count = redis_client.incr(window_key)

            # Set expiration for window
            redis_client.expire(window_key, 60)  # 1 minute

            return count <= rate_limit

        except Exception as e:
            logger.error(f"Error checking rate limit: {e}")
            return True  # Allow on error

    def revoke_api_key(self, key_id: str, user_id: str) -> bool:
        """Revoke an API key."""
        try:
            # Get key record
            key_data = redis_client.get(f"api_key:{key_id}")
            if not key_data:
                return False

            key_dict = eval(key_data)
            if key_dict["user_id"] != user_id:
                return False  # Not owner's key

            # Mark as suspended
            key_dict["status"] = "suspended"
            redis_client.setex(
                f"api_key:{key_id}",
                30 * 24 * 60 * 60,  # Keep for 30 days
                str(key_dict),
            )

            logger.info(f"Revoked API key {key_id}")
            return True

        except Exception as e:
            logger.error(f"Error revoking API key: {e}")
            return False

    def list_user_keys(self, user_id: str) -> list[APIKey]:
        """List all API keys for a user."""
        try:
            key_ids = redis_client.smembers(f"user_keys:{user_id}")
            keys = []

            for key_id in key_ids:
                key_data = redis_client.get(f"api_key:{key_id}")
                if key_data:
                    key_dict = eval(key_data)
                    keys.append(APIKey(**key_dict))

            return keys

        except Exception as e:
            logger.error(f"Error listing user keys: {e}")
            return []

    def _hash_key(self, api_key: str) -> str:
        """Hash API key for storage."""
        return hashlib.sha256(api_key.encode()).hexdigest()

    def _update_key_record(self, key: APIKey):
        """Update key record in Redis."""
        try:
            redis_client.setex(
                f"api_key:{key.key_id}",
                365 * 24 * 60 * 60,  # 1 year
                str(key.__dict__),
            )
        except Exception as e:
            logger.error(f"Error updating key record: {e}")


# Global instance
api_key_service = APIKeyService()
