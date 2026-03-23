#!/usr/bin/env bash

set -euo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <custom-host-url>" >&2
    echo "Example: $0 https://test.eth-rpc.status.im" >&2
    exit 1
fi

BASE_URL="$1"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="$SCRIPT_DIR/.venv_get_proxy_token"
TEMP_DIR="$(mktemp -d)"

cleanup() {
    if [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}
trap cleanup EXIT

if [ ! -d "$VENV_DIR" ]; then
    python3 -m venv "$VENV_DIR" >/dev/null 2>&1
    "$VENV_DIR/bin/pip" install --quiet argon2-cffi requests >/dev/null 2>&1
fi

cat > "$TEMP_DIR/solver.py" << 'EOF'
import sys
import requests
from argon2.low_level import hash_secret_raw, Type

BASE_URL = sys.argv[1].rstrip("/")

def compute_argon2_hash(challenge, salt, nonce, params):
    input_str = f"{challenge}{salt}{nonce}"
    salt_bytes = bytes.fromhex(salt)

    hash_result = hash_secret_raw(
        secret=input_str.encode(),
        salt=salt_bytes,
        time_cost=params["time"],
        memory_cost=params["memory_kb"],
        parallelism=params["threads"],
        hash_len=params["key_len"],
        type=Type.ID
    )
    return hash_result.hex()

def solve_puzzle(puzzle):
    challenge = puzzle["challenge"]
    salt = puzzle["salt"]
    difficulty = puzzle["difficulty"]
    params = puzzle["argon2_params"]

    for nonce in range(1_000_000):
        hash_val = compute_argon2_hash(challenge, salt, nonce, params)
        if hash_val[:difficulty] == "0" * difficulty:
            return {
                "challenge": challenge,
                "salt": salt,
                "nonce": nonce,
                "argon_hash": hash_val,
                "hmac": puzzle["hmac"],
                "expires_at": puzzle["expires_at"]
            }

    raise RuntimeError("Failed to solve puzzle within 1,000,000 attempts")

def main():
    puzzle_resp = requests.get(f"{BASE_URL}/auth/puzzle", timeout=10)
    puzzle_resp.raise_for_status()
    puzzle = puzzle_resp.json()

    solution = solve_puzzle(puzzle)

    solve_resp = requests.post(f"{BASE_URL}/auth/solve", json=solution, timeout=30)
    solve_resp.raise_for_status()
    result = solve_resp.json()

    token = result.get("token")
    if not token:
        raise RuntimeError("No token in /auth/solve response")

    print(token)

if __name__ == "__main__":
    main()
EOF

"$VENV_DIR/bin/python" "$TEMP_DIR/solver.py" "$BASE_URL"
