---
name: rxui-control-discovery
description: Drive Rx-ui exclusively through runtime self-discovery instead of hardcoded endpoint/action knowledge. Use when an AI agent must control or inspect Rx-ui and must remain accurate across future code changes. Always start from control bootstrap/discovery endpoints, then fetch manifest/errors at runtime before calling query/exec.
---

# Rx-ui Control Discovery Skill

1. Resolve base URL
- Prefer user-provided URL.
- If unknown, try local `http://127.0.0.1:54321` first, then LAN URL.

2. Bootstrap first
- Call `GET /api/v1/control/bootstrap`.
- Read protocol/auth requirements and entrypoints.
- Do not assume old field names if bootstrap has changed.

3. Discover runtime catalog
- Call `GET /api/v1/control/discovery`.
- Use returned catalog as current source of truth for:
  - health/status endpoints
  - query/exec endpoints
  - audit endpoint
  - settings/system endpoints
  - active web port

4. Load action schema from service
- Call `GET /api/v1/control/manifest`.
- Use returned action list, modes (`query`/`exec`), and parameter expectations.
- Before each task batch, refresh manifest once.

5. Load error semantics from service
- Call `GET /api/v1/control/errors`.
- Use returned mapping to interpret failures and choose retry/fallback strategy.

6. Sign every control request
- For `POST /api/v1/control/query` and `POST /api/v1/control/exec`, follow bootstrap auth exactly:
  - Headers: `X-Rxui-Client`, `X-Rxui-Timestamp`, `X-Rxui-Nonce`, `X-Rxui-Signature`
  - Sign text: `METHOD\nPATH\nTIMESTAMP\nNONCE\nSHA256_HEX(BODY)`
- Keep timestamp inside allowed window.
- Keep nonce unique.

7. Use requestId for idempotency
- Always include `requestId` for non-trivial operations.
- If response indicates idempotency hit, treat it as successful replay unless payload says otherwise.

8. Verify and audit
- For write operations, run a follow-up query to verify state.
- Optionally read `GET /api/v1/control/audit` to confirm execution trail.

9. Robustness rules
- Never hardcode action semantics in prompts or code when manifest is available.
- Never rely on repository markdown as runtime truth.
- If discovery/manifest/errors endpoints are unavailable, report degraded mode explicitly and stop dangerous writes.
