FROM python:3.13-slim AS build
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/
WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev

FROM python:3.13-slim
WORKDIR /app
COPY --from=build /app/.venv ./.venv
COPY src/ ./src/
ENV PATH="/app/.venv/bin:$PATH"
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8000"]
