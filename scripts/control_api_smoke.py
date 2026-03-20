#!/usr/bin/env python3
import base64
import hashlib
import json
import os
import time
from dataclasses import dataclass

import requests
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives import serialization

BASE = os.environ.get("RXUI_BASE", "http://127.0.0.1:54321")
API = BASE + "/api/v1"

@dataclass
class Ctx:
    client_id: str
    sk: Ed25519PrivateKey


def jprint(obj):
    print(json.dumps(obj, ensure_ascii=False, indent=2))


def must(cond, msg):
    if not cond:
        raise SystemExit(f"[FAIL] {msg}")


def sign_headers(ctx: Ctx, path: str, body: dict):
    body_bytes = json.dumps(body, separators=(",", ":")).encode()
    ts = str(int(time.time()))
    nonce = f"n{int(time.time()*1000)}"
    h = hashlib.sha256(body_bytes).hexdigest()
    text = "\n".join(["POST", path, ts, nonce, h]).encode()
    sig = ctx.sk.sign(text)
    return {
        "X-Rxui-Client": ctx.client_id,
        "X-Rxui-Timestamp": ts,
        "X-Rxui-Nonce": nonce,
        "X-Rxui-Signature": base64.b64encode(sig).decode(),
        "Content-Type": "application/json",
    }, body_bytes


def call_signed(ctx: Ctx, path: str, body: dict):
    headers, body_bytes = sign_headers(ctx, path, body)
    r = requests.post(BASE + path, headers=headers, data=body_bytes, timeout=20)
    return r


def main():
    print("[1] bootstrap")
    b = requests.get(API + "/control/bootstrap", timeout=10)
    must(b.status_code == 200, "bootstrap http")
    bj = b.json()
    must(bj.get("code") == 0, "bootstrap code")

    print("[2] register control client")
    sk = Ed25519PrivateKey.generate()
    pk = sk.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    )
    client_id = f"smoke-{int(time.time())}"
    reg = requests.post(API + "/control/clients", json={
        "clientId": client_id,
        "publicKey": base64.b64encode(pk).decode(),
        "enabled": True,
        "remark": "smoke",
    }, timeout=10)
    must(reg.status_code == 200 and reg.json().get("code") == 0, "register client")

    ctx = Ctx(client_id=client_id, sk=sk)

    print("[3] query xray.status")
    q = call_signed(ctx, "/api/v1/control/query", {
        "requestId": "smoke-q-1",
        "action": "xray.status",
        "params": {}
    })
    must(q.status_code == 200, "query http")
    qj = q.json()
    must(qj.get("ok") is True, f"query ok: {qj}")

    print("[4] idempotency check")
    body = {"requestId": "smoke-q-dup", "action": "sys.status", "params": {}}
    q1 = call_signed(ctx, "/api/v1/control/query", body).json()
    q2 = call_signed(ctx, "/api/v1/control/query", body).json()
    must(q1.get("ok") is True and q2.get("ok") is True, "idempotency responses")
    must(q2.get("idempotencyHit") is True, "idempotency hit flag")

    print("[5] exec net.ping")
    e = call_signed(ctx, "/api/v1/control/exec", {
        "requestId": "smoke-e-1",
        "action": "net.ping",
        "params": {"target": "127.0.0.1"}
    })
    ej = e.json()
    must(e.status_code == 200 and ej.get("ok") is True, f"exec ping: {ej}")

    print("[6] audit exists")
    a = requests.get(API + "/control/audit", params={"clientId": client_id, "limit": 20}, timeout=10)
    must(a.status_code == 200 and a.json().get("code") == 0, "audit query")
    rows = a.json().get("data") or []
    must(len(rows) >= 3, "audit row count")

    print("[PASS] control api smoke test passed")


if __name__ == "__main__":
    main()
