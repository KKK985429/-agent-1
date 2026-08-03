"""Request-scoped API-key handling for the WeKnora MCP transports."""

import contextvars


_request_api_key = contextvars.ContextVar("weknora_request_api_key", default="")


def resolve_api_key(configured_api_key: str) -> str:
    """Prefer the deployment key, falling back to the current MCP request."""
    api_key = configured_api_key.strip() or _request_api_key.get().strip()
    if not api_key:
        raise ValueError(
            "WeKnora API key is required: configure WEKNORA_API_KEY or send X-API-Key"
        )
    return api_key


class APIKeyContextMiddleware:
    """Expose the inbound MCP X-API-Key to tool calls for this request only."""

    def __init__(self, asgi_app):
        self.asgi_app = asgi_app

    async def __call__(self, scope, receive, send):
        api_key = ""
        if scope.get("type") in {"http", "websocket"}:
            for name, value in scope.get("headers", []):
                if name.lower() == b"x-api-key":
                    api_key = value.decode("latin-1").strip()
                    break

        context_token = _request_api_key.set(api_key)
        try:
            await self.asgi_app(scope, receive, send)
        finally:
            _request_api_key.reset(context_token)
