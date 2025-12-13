"""
HelixFlow RBAC Authorization Framework
Role-Based Access Control with fine-grained permissions
"""

from typing import List, Dict, Set, Optional
from dataclasses import dataclass
from enum import Enum
import json


class Permission(Enum):
    # API Gateway permissions
    API_ACCESS = "api:access"
    API_RATE_LIMIT_BYPASS = "api:rate_limit_bypass"

    # Model permissions
    MODEL_LIST = "model:list"
    MODEL_INFERENCE = "model:inference"
    MODEL_ADMIN = "model:admin"

    # User permissions
    USER_READ = "user:read"
    USER_UPDATE = "user:update"
    USER_ADMIN = "user:admin"

    # Billing permissions
    BILLING_READ = "billing:read"
    BILLING_UPDATE = "billing:update"
    BILLING_ADMIN = "billing:admin"

    # Admin permissions
    SYSTEM_ADMIN = "system:admin"
    MONITORING_READ = "monitoring:read"
    MONITORING_ADMIN = "monitoring:admin"


@dataclass
class Role:
    name: str
    description: str
    permissions: Set[Permission]
    inherits_from: Optional[List[str]] = None


class RBACService:
    def __init__(self):
        self.roles = self._initialize_roles()
        self.user_roles = {}  # user_id -> set of role names
        self.role_permissions_cache = {}

    def _initialize_roles(self) -> Dict[str, Role]:
        """Initialize default roles with their permissions"""
        return {
            "free": Role(
                name="free",
                description="Basic user with limited access",
                permissions={
                    Permission.API_ACCESS,
                    Permission.MODEL_LIST,
                    Permission.MODEL_INFERENCE,
                    Permission.USER_READ,
                    Permission.BILLING_READ,
                },
            ),
            "pro": Role(
                name="pro",
                description="Professional user with enhanced access",
                permissions={
                    Permission.API_ACCESS,
                    Permission.API_RATE_LIMIT_BYPASS,
                    Permission.MODEL_LIST,
                    Permission.MODEL_INFERENCE,
                    Permission.USER_READ,
                    Permission.USER_UPDATE,
                    Permission.BILLING_READ,
                    Permission.BILLING_UPDATE,
                },
            ),
            "enterprise": Role(
                name="enterprise",
                description="Enterprise user with advanced features",
                permissions={
                    Permission.API_ACCESS,
                    Permission.API_RATE_LIMIT_BYPASS,
                    Permission.MODEL_LIST,
                    Permission.MODEL_INFERENCE,
                    Permission.MODEL_ADMIN,
                    Permission.USER_READ,
                    Permission.USER_UPDATE,
                    Permission.USER_ADMIN,
                    Permission.BILLING_READ,
                    Permission.BILLING_UPDATE,
                    Permission.BILLING_ADMIN,
                    Permission.MONITORING_READ,
                },
            ),
            "research": Role(
                name="research",
                description="Research user with full access",
                permissions=set(Permission),  # All permissions
                inherits_from=["enterprise"],
            ),
            "admin": Role(
                name="admin",
                description="System administrator with full access",
                permissions=set(Permission),  # All permissions
            ),
        }

    def assign_role_to_user(self, user_id: str, role_name: str):
        """Assign a role to a user"""
        if role_name not in self.roles:
            raise ValueError(f"Role {role_name} does not exist")

        if user_id not in self.user_roles:
            self.user_roles[user_id] = set()

        self.user_roles[user_id].add(role_name)
        # Clear permission cache for this user
        self.role_permissions_cache.pop(user_id, None)

    def remove_role_from_user(self, user_id: str, role_name: str):
        """Remove a role from a user"""
        if user_id in self.user_roles:
            self.user_roles[user_id].discard(role_name)
            # Clear permission cache for this user
            self.role_permissions_cache.pop(user_id, None)

    def get_user_permissions(self, user_id: str) -> Set[Permission]:
        """Get all permissions for a user based on their roles"""
        if user_id in self.role_permissions_cache:
            return self.role_permissions_cache[user_id]

        permissions = set()
        user_role_names = self.user_roles.get(user_id, set())

        for role_name in user_role_names:
            if role_name in self.roles:
                role = self.roles[role_name]
                permissions.update(role.permissions)

                # Add inherited permissions
                if role.inherits_from:
                    for inherited_role in role.inherits_from:
                        if inherited_role in self.roles:
                            permissions.update(self.roles[inherited_role].permissions)

        # Cache the result
        self.role_permissions_cache[user_id] = permissions
        return permissions

    def check_permission(self, user_id: str, permission: Permission) -> bool:
        """Check if user has a specific permission"""
        user_permissions = self.get_user_permissions(user_id)
        return permission in user_permissions

    def check_permissions(self, user_id: str, permissions: List[Permission]) -> bool:
        """Check if user has all specified permissions"""
        user_permissions = self.get_user_permissions(user_id)
        return all(perm in user_permissions for perm in permissions)

    def require_permission(self, user_id: str, permission: Permission):
        """Raise exception if user doesn't have permission"""
        if not self.check_permission(user_id, permission):
            raise PermissionDeniedError(
                f"User {user_id} lacks permission: {permission.value}"
            )

    def require_permissions(self, user_id: str, permissions: List[Permission]):
        """Raise exception if user doesn't have all permissions"""
        if not self.check_permissions(user_id, permissions):
            missing = [
                p.value for p in permissions if not self.check_permission(user_id, p)
            ]
            raise PermissionDeniedError(f"User {user_id} lacks permissions: {missing}")

    def get_user_roles(self, user_id: str) -> Set[str]:
        """Get all role names for a user"""
        return self.user_roles.get(user_id, set()).copy()

    def create_custom_role(
        self,
        name: str,
        description: str,
        permissions: Set[Permission],
        inherits_from: Optional[List[str]] = None,
    ):
        """Create a custom role (for enterprise customers)"""
        if name in self.roles:
            raise ValueError(f"Role {name} already exists")

        self.roles[name] = Role(
            name=name,
            description=description,
            permissions=permissions,
            inherits_from=inherits_from or [],
        )

    def get_role_details(self, role_name: str) -> Optional[Dict]:
        """Get detailed information about a role"""
        if role_name not in self.roles:
            return None

        role = self.roles[role_name]
        return {
            "name": role.name,
            "description": role.description,
            "permissions": [p.value for p in role.permissions],
            "inherits_from": role.inherits_from or [],
        }


class PermissionDeniedError(Exception):
    """Raised when a user lacks required permissions"""

    pass
