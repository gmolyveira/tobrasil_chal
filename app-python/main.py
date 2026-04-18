from datetime import datetime, timezone

from fastapi import FastAPI
from prometheus_fastapi_instrumentator import Instrumentator

app = FastAPI(title="App Python")
Instrumentator().instrument(app).expose(app)


@app.get("/")
def root():
    return {"message": "Hello from Python (FastAPI)"}


@app.get("/time")
def current_time():
    now = datetime.now(timezone.utc).isoformat()
    return {"server_time": now}
