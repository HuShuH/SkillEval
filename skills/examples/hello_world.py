"""Minimal hello_world skill for the phase two MVP."""

from __future__ import annotations

import json


def main() -> None:
    """Emit a simple JSON result for runner validation."""
    print(json.dumps({"status": "success", "message": "Hello from skill"}))


if __name__ == "__main__":
    main()
