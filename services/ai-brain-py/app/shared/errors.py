"""Galat bisnis (padanan apperror di core-api). Service melempar ini; observability.py mengubahnya jadi
JSON {"error": "..."} dengan kode HTTP yang sesuai. Service tidak perlu tahu soal HTTP."""


class AppError(Exception):
    status = 400

    def __init__(self, message: str) -> None:
        super().__init__(message)
        self.message = message


class Invalid(AppError):
    status = 400


class Forbidden(AppError):
    status = 403


class NotFound(AppError):
    """Juga dipakai untuk "ada tapi tidak boleh dilihat": tidak membocorkan keberadaan dokumen."""

    status = 404


class Conflict(AppError):
    status = 409


class TooLarge(AppError):
    status = 413
