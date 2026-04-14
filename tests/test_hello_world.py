from __future__ import annotations

import sys
from pathlib import Path

from runners.basic_runner import BasicRunner


PROJECT_ROOT = Path(__file__).resolve().parents[1]
HELLO_WORLD_SKILL = PROJECT_ROOT / "skills" / "examples" / "hello_world.py"


def test_hello_world_skill_success() -> None:
    runner = BasicRunner(timeout_seconds=5)

    result = runner.run(HELLO_WORLD_SKILL)

    assert result["status"] == "success"
    assert result["output"] == {"status": "success", "message": "Hello from skill"}
    assert result["error"] is None


def test_hello_world_runner_metadata() -> None:
    runner = BasicRunner(timeout_seconds=5)

    result = runner.run(HELLO_WORLD_SKILL)
    metadata = result["metadata"]

    assert metadata["schema_loaded"] is True
    assert metadata["timeout_seconds"] == 5
    assert metadata["timed_out"] is False
    assert metadata["return_code"] == 0
    assert metadata["duration_seconds"] >= 0
    assert metadata["output_parse_error"] is None


def test_runner_timeout_with_temp_skill(tmp_path: Path) -> None:
    slow_skill = tmp_path / "slow_skill.py"
    slow_skill.write_text(
        "import time\ntime.sleep(2)\nprint('{\"status\": \"success\"}')\n",
        encoding="utf-8",
    )
    runner = BasicRunner(timeout_seconds=1)

    result = runner.run(slow_skill)

    assert result["status"] == "timeout"
    assert result["metadata"]["timed_out"] is True
    assert result["metadata"]["timeout_seconds"] == 1
    assert result["metadata"]["return_code"] is None


def test_runner_handles_invalid_json_output(tmp_path: Path) -> None:
    invalid_json_skill = tmp_path / "invalid_json_skill.py"
    invalid_json_skill.write_text(
        "print('not json')\n",
        encoding="utf-8",
    )
    runner = BasicRunner(timeout_seconds=5)

    result = runner.run(invalid_json_skill)

    assert result["status"] == "success"
    assert result["output"] is None
    assert result["raw_stdout"].strip() == "not json"
    assert result["metadata"]["output_parse_error"] is not None
