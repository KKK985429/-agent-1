import unittest

from mcp_request_auth import APIKeyContextMiddleware, resolve_api_key


class APIKeyContextMiddlewareTest(unittest.IsolatedAsyncioTestCase):
    async def test_request_header_is_used_when_configured_key_is_empty(self):
        observed = []

        async def app(_scope, _receive, _send):
            observed.append(resolve_api_key(""))

        middleware = APIKeyContextMiddleware(app)
        await middleware(
            {"type": "http", "headers": [(b"x-api-key", b"tenant-key")]},
            None,
            None,
        )

        self.assertEqual(["tenant-key"], observed)
        with self.assertRaises(ValueError):
            resolve_api_key("")

    async def test_configured_key_takes_precedence(self):
        observed = []

        async def app(_scope, _receive, _send):
            observed.append(resolve_api_key("configured-key"))

        middleware = APIKeyContextMiddleware(app)
        await middleware(
            {"type": "http", "headers": [(b"X-API-KEY", b"request-key")]},
            None,
            None,
        )

        self.assertEqual(["configured-key"], observed)


if __name__ == "__main__":
    unittest.main()
