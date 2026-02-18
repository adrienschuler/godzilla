from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .database import db
from .routes import router


@asynccontextmanager
async def lifespan(app: FastAPI):
    await db.discussions.create_index([("users", 1), ("updated_at", -1)])
    await db.messages.create_index([("discussion_id", 1), ("_id", -1)])
    yield


app = FastAPI(title="History Service", version="1.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router)


@app.get("/health")
async def health():
    return {"status": "healthy"}
