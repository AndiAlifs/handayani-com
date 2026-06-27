"""Validates the JWT issued by the NYAMPE Go service.

FastAPI does NOT mint tokens or store users — the Go service (proxied via
routers/gateway.py) owns authentication. This module only *verifies* the
incoming Bearer token: HS256, signed with the shared JWT_SECRET. The Go token
carries `user_id` and `role` claims (roles: employee | manager | instructor).
`manager` is the elevated/admin-equivalent role.
"""
import os

import jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from pydantic import BaseModel

_ALGORITHM = "HS256"
# Matches the Go service's dev fallback so local dev "just works" with no .env.
_DEFAULT_SECRET = "super-secret-key-default"
_bearer = HTTPBearer(auto_error=False)


class TokenUser(BaseModel):
    userId: int
    role: str


def _secret() -> str:
    return os.getenv("JWT_SECRET", _DEFAULT_SECRET)


def decode_token(token: str) -> "TokenUser":
    try:
        payload = jwt.decode(token, _secret(), algorithms=[_ALGORITHM])
        # Parse the claim inside the guard: a validly-signed token with a
        # non-numeric user_id must fail closed as 401, not bubble up as a 500.
        user_id = int(payload.get("user_id", 0))
    except (jwt.PyJWTError, ValueError, TypeError):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid or expired token"
        )
    return TokenUser(userId=user_id, role=payload.get("role", ""))


def require_auth(creds: HTTPAuthorizationCredentials = Depends(_bearer)) -> "TokenUser":
    """Any authenticated NYAMPE user."""
    if creds is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED, detail="Not authenticated"
        )
    return decode_token(creds.credentials)


def require_manager(user: "TokenUser" = Depends(require_auth)) -> "TokenUser":
    """Manager-only endpoints (the admin-equivalent role)."""
    if user.role != "manager":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN, detail="Manager role required"
        )
    return user
