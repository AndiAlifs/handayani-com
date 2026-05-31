"""Reverse-proxy router: forwards auth/attendance/admin/instructor calls to the
vendored Go (NYAMPE) service. The browser only ever talks to FastAPI."""
import os
from typing import Optional

import httpx
from fastapi import APIRouter, Request, Response

router = APIRouter(tags=["gateway"])

GO_BACKEND_URL = os.getenv("GO_BACKEND_URL", "http://localhost:8090").rstrip("/")
_TIMEOUT = httpx.Timeout(15.0)

# Frontend prefix -> Go path prefix.
_PREFIX_MAP = {
    "/api/auth": "/api",          # /api/auth/login   -> /api/login
    "/api/attendance": "/api",    # /api/attendance/clock-in -> /api/clock-in
    "/api/admin": "/api/admin",   # passthrough
    "/api/instructor": "/api/instructor",
}


def _target(path: str) -> Optional[str]:
    for prefix, go_prefix in _PREFIX_MAP.items():
        if path == prefix or path.startswith(prefix + "/"):
            return go_prefix + path[len(prefix):]
    return None


async def _proxy(request: Request, path: str) -> Response:
    go_path = _target(path)
    if go_path is None:
        return Response(status_code=404)
    url = GO_BACKEND_URL + go_path
    body = await request.body()
    headers = {
        k: v for k, v in request.headers.items()
        if k.lower() in ("authorization", "content-type", "accept")
    }
    try:
        async with httpx.AsyncClient(timeout=_TIMEOUT) as cx:
            resp = await cx.request(
                request.method, url, params=request.query_params,
                content=body, headers=headers,
            )
    except httpx.HTTPError:
        return Response(
            content='{"error": "Layanan absensi tidak tersedia"}',
            status_code=502, media_type="application/json",
        )
    return Response(
        content=resp.content, status_code=resp.status_code,
        media_type=resp.headers.get("content-type"),
    )


@router.api_route(
    "/api/auth/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_auth(path: str, request: Request):
    return await _proxy(request, request.url.path)


@router.api_route(
    "/api/attendance/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_attendance(path: str, request: Request):
    return await _proxy(request, request.url.path)


@router.api_route(
    "/api/admin/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_admin(path: str, request: Request):
    return await _proxy(request, request.url.path)


@router.api_route(
    "/api/instructor/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_instructor(path: str, request: Request):
    return await _proxy(request, request.url.path)
