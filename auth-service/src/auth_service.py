"""
HelixFlow Authentication Service
JWT-based authentication with RS256 signatures and RBAC
"""

import os
import jwt
import bcrypt
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
from dataclasses import dataclass
import redis
import logging


@dataclass
class User:
    user_id: str
    email: str
    subscription_tier: str
    api_key_hash: str
    status: str


@dataclass
class TokenPair:
    access_token: str
    refresh_token: str
    expires_in: int


class AuthService:
    def __init__(self):
        self.redis_client = redis.Redis(
            host=os.getenv("REDIS_HOST", "localhost"),
            port=int(os.getenv("REDIS_PORT", 6379)),
            password=os.getenv("REDIS_PASSWORD"),
            decode_responses=True,
        )

        # Load RSA keys
        with open("/secrets/jwt-private.pem", "r") as f:
            self.private_key = f.read()
        with open("/secrets/jwt-public.pem", "r") as f:
            self.public_key = f.read()

        self.logger = logging.getLogger(__name__)

    def authenticate_user(self, email: str, password: str) -> Optional[User]:
        """Authenticate user with email and password"""
        try:
            # Get user from database (placeholder - integrate with actual DB)
            user_data = self._get_user_by_email(email)
            if not user_data:
                return None

            # Verify password
            if not bcrypt.checkpw(
                password.encode(), user_data["api_key_hash"].encode()
            ):
                return None

            return User(**user_data)
        except Exception as e:
            self.logger.error(f"Authentication failed for {email}: {e}")
            return None

    def generate_tokens(self, user: User) -> TokenPair:
        """Generate access and refresh tokens"""
        now = datetime.utcnow()

        # Access token (short-lived)
        access_payload = {
            "sub": user.user_id,
            "email": user.email,
            "tier": user.subscription_tier,
            "type": "access",
            "iat": now,
            "exp": now + timedelta(minutes=15),
        }

        access_token = jwt.encode(access_payload, self.private_key, algorithm="RS256")

        # Refresh token (long-lived)
        refresh_payload = {
            "sub": user.user_id,
            "type": "refresh",
            "iat": now,
            "exp": now + timedelta(days=30),
        }

        refresh_token = jwt.encode(refresh_payload, self.private_key, algorithm="RS256")

        # Store refresh token in Redis
        self.redis_client.setex(
            f"refresh_token:{user.user_id}",
            30 * 24 * 60 * 60,  # 30 days
            refresh_token,
        )

        return TokenPair(
            access_token=access_token,
            refresh_token=refresh_token,
            expires_in=15 * 60,  # 15 minutes
        )

    def validate_token(
        self, token: str, token_type: str = "access"
    ) -> Optional[Dict[str, Any]]:
        """Validate JWT token"""
        try:
            payload = jwt.decode(
                token,
                self.public_key,
                algorithms=["RS256"],
                options={"verify_exp": True},
            )

            if payload.get("type") != token_type:
                return None

            return payload
        except jwt.ExpiredSignatureError:
            self.logger.warning("Token expired")
            return None
        except jwt.InvalidTokenError as e:
            self.logger.warning(f"Invalid token: {e}")
            return None

    def refresh_access_token(self, refresh_token: str) -> Optional[TokenPair]:
        """Generate new access token using refresh token"""
        try:
            payload = self.validate_token(refresh_token, "refresh")
            if not payload:
                return None

            user_id = payload["sub"]

            # Verify refresh token exists in Redis
            stored_token = self.redis_client.get(f"refresh_token:{user_id}")
            if not stored_token or stored_token != refresh_token:
                return None

            # Get user data (placeholder)
            user_data = self._get_user_by_id(user_id)
            if not user_data:
                return None

            user = User(**user_data)
            return self.generate_tokens(user)

        except Exception as e:
            self.logger.error(f"Token refresh failed: {e}")
            return None

    def revoke_token(self, user_id: str, token_type: str = "refresh"):
        """Revoke refresh token"""
        self.redis_client.delete(f"{token_type}_token:{user_id}")

    def authorize_request(
        self, token_payload: Dict[str, Any], required_tier: str = None
    ) -> bool:
        """Check if user has required permissions"""
        user_tier = token_payload.get("tier", "FREE")

        tier_hierarchy = {"FREE": 0, "PRO": 1, "ENTERPRISE": 2, "RESEARCH": 3}

        if required_tier and tier_hierarchy.get(user_tier, 0) < tier_hierarchy.get(
            required_tier, 0
        ):
            return False

        return True

    def _get_user_by_email(self, email: str) -> Optional[Dict[str, Any]]:
        """Get user by email (placeholder - integrate with database)"""
        # This would query the actual database
        return {
            "user_id": "user-123",
            "email": email,
            "subscription_tier": "PRO",
            "api_key_hash": bcrypt.hashpw(
                "password123".encode(), bcrypt.gensalt()
            ).decode(),
            "status": "ACTIVE",
        }

    def _get_user_by_id(self, user_id: str) -> Optional[Dict[str, Any]]:
        """Get user by ID (placeholder - integrate with database)"""
        # This would query the actual database
        return {
            "user_id": user_id,
            "email": "user@example.com",
            "subscription_tier": "PRO",
            "api_key_hash": bcrypt.hashpw(
                "password123".encode(), bcrypt.gensalt()
            ).decode(),
            "status": "ACTIVE",
        }
