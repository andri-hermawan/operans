"""Pengamatan: log satu baris JSON (sama gaya dengan service Go) + penanda request_id di setiap permintaan."""
import json
import logging
import time
import uuid

from fastapi import FastAPI, Request
from starlette.exceptions import HTTPException  # induk HTTPException FastAPI: juga menangkap 404/405 bawaan
from fastapi.responses import JSONResponse

from app.shared.errors import AppError

log = logging.getLogger("ai-brain")


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        line = {
            "time": self.formatTime(record, "%Y-%m-%dT%H:%M:%S%z"),
            "level": record.levelname,
            "msg": record.getMessage(),
            **getattr(record, "extra_fields", {}),
        }
        if record.exc_info:
            line["error"] = self.formatException(record.exc_info)
        return json.dumps(line, ensure_ascii=False)


def setup_logging(level: str) -> None:
    handler = logging.StreamHandler()
    handler.setFormatter(JsonFormatter())
    root = logging.getLogger()
    root.handlers = [handler]
    root.setLevel(level.upper())


def install_http_basics(app: FastAPI) -> None:
    """Request-ID + log akses, dan bentuk galat seragam {"error": "..."} seperti service Go."""

    @app.middleware("http")
    async def request_log(request: Request, call_next):
        request_id = request.headers.get("X-Request-ID") or uuid.uuid4().hex
        request.state.request_id = request_id
        started = time.perf_counter()
        response = await call_next(request)
        response.headers["X-Request-ID"] = request_id
        log.info(
            "request",
            extra={"extra_fields": {
                "request_id": request_id, "method": request.method, "path": request.url.path,
                "status": response.status_code, "ms": round((time.perf_counter() - started) * 1000, 1),
            }},
        )
        return response

    @app.exception_handler(HTTPException)
    async def http_error(_: Request, exc: HTTPException):
        return JSONResponse({"error": exc.detail}, status_code=exc.status_code)

    @app.exception_handler(AppError)
    async def app_error(_: Request, exc: AppError):
        return JSONResponse({"error": exc.message}, status_code=exc.status)
