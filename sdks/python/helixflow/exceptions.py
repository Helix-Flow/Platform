"""
HelixFlow Exceptions
"""

class HelixFlowError(Exception):
    """Base exception for HelixFlow errors."""
    pass

class AuthenticationError(HelixFlowError):
    """Raised when authentication fails."""
    pass

class RateLimitError(HelixFlowError):
    """Raised when rate limit is exceeded."""
    pass

class APIError(HelixFlowError):
    """Raised when API returns an error."""
    pass
