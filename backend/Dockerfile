# FastAPI backend (Python 3.12). Config comes from environment (see compose.yml).
FROM python:3.12-slim

WORKDIR /app

# cryptography wheel covers MySQL's caching_sha2 if ever needed; install deps first
# so the layer caches across code changes.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
