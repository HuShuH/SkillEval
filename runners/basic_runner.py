"""Basic subprocess-based skill runner for the phase two MVP."""

from __future__ import annotations

import json
import subprocess
import sys
import time
from datetime import UTC, datetime
from pathlib import Path
from typing import Any


class BasicRunner:
    """Run Python skill scripts with timeout and standardized JSON results."""

    def __init__(self, schema_path: str | Path | None = None, timeout_seconds: int = 10) -> None:
        if timeout_seconds < 1:
            raise ValueError("timeout_seconds must be >= 1")

        self.project_root = Path(__file__).resolve().parents[1]
        self.schema_path = Path(schema_path) if schema_path else self.project_root / "schemas" / "skill_schema.json"
        self.timeout_seconds = timeout_seconds
        self.schema: dict[str, Any] | None = None

    def load_schema(self) -> dict[str, Any]:
        """Load the skill definition schema from disk."""
        with self.schema_path.open("r", encoding="utf-8") as schema_file:
            self.schema = json.load(schema_file)
        return self.schema

    def run(self, skill_path: str | Path, parameters: dict[str, Any] | None = None) -> dict[str, Any]:
        """Execute a skill and return a standardized JSON-serializable result."""
        return self.run_skill(skill_path=skill_path, parameters=parameters)

    def run_skill(self, skill_path: str | Path, parameters: dict[str, Any] | None = None) -> dict[str, Any]:
        """Execute a Python skill script via subprocess with timeout control."""
        started_at = self._utc_now()
        start_time = time.perf_counter()
        resolved_skill_path = self._resolve_skill_path(skill_path)

        try:
            self.load_schema()
        except (OSError, json.JSONDecodeError) as exc:
            return self._error_result(
                status="error",
                skill_path=resolved_skill_path,
                started_at=started_at,
                start_time=start_time,
                error=f"Failed to load schema: {exc}",
                parameters=parameters,
            )

        if not resolved_skill_path.is_file():
            return self._error_result(
                status="error",
                skill_path=resolved_skill_path,
                started_at=started_at,
                start_time=start_time,
                error=f"Skill file not found: {resolved_skill_path}",
                parameters=parameters,
            )

        command = self._build_command(resolved_skill_path, parameters)

        try:
            completed = subprocess.run(
                command,
                capture_output=True,
                text=True,
                timeout=self.timeout_seconds,
                check=False,
            )
        except subprocess.TimeoutExpired as exc:
            ended_at = self._utc_now()
            stdout = self._coerce_output(exc.stdout)
            stderr = self._coerce_output(exc.stderr)
            return self._result(
                status="timeout",
                skill_path=resolved_skill_path,
                started_at=started_at,
                ended_at=ended_at,
                duration_seconds=time.perf_counter() - start_time,
                return_code=None,
                stdout=stdout,
                stderr=stderr,
                output=None,
                error=f"Skill execution timed out after {self.timeout_seconds} seconds",
                timed_out=True,
                parameters=parameters,
                output_parse_error=None,
            )

        ended_at = self._utc_now()
        output, output_parse_error = self._parse_stdout(completed.stdout)
        status = "success" if completed.returncode == 0 else "failed"

        return self._result(
            status=status,
            skill_path=resolved_skill_path,
            started_at=started_at,
            ended_at=ended_at,
            duration_seconds=time.perf_counter() - start_time,
            return_code=completed.returncode,
            stdout=completed.stdout,
            stderr=completed.stderr,
            output=output,
            error=None if status == "success" else "Skill process exited with a non-zero return code",
            timed_out=False,
            parameters=parameters,
            output_parse_error=output_parse_error,
        )

    def _build_command(self, skill_path: Path, parameters: dict[str, Any] | None) -> list[str]:
        command = [sys.executable, str(skill_path)]
        if parameters is not None:
            command.extend(["--input", json.dumps(parameters, ensure_ascii=False)])
        return command

    def _parse_stdout(self, stdout: str) -> tuple[Any | None, str | None]:
        stripped_stdout = stdout.strip()
        if not stripped_stdout:
            return None, None

        try:
            return json.loads(stripped_stdout), None
        except json.JSONDecodeError as exc:
            return None, f"Failed to parse stdout as JSON: {exc}"

    def _resolve_skill_path(self, skill_path: str | Path) -> Path:
        path = Path(skill_path)
        if path.is_absolute():
            return path
        return self.project_root / path

    def _error_result(
        self,
        *,
        status: str,
        skill_path: Path,
        started_at: str,
        start_time: float,
        error: str,
        parameters: dict[str, Any] | None,
    ) -> dict[str, Any]:
        ended_at = self._utc_now()
        return self._result(
            status=status,
            skill_path=skill_path,
            started_at=started_at,
            ended_at=ended_at,
            duration_seconds=time.perf_counter() - start_time,
            return_code=None,
            stdout="",
            stderr="",
            output=None,
            error=error,
            timed_out=False,
            parameters=parameters,
            output_parse_error=None,
        )

    def _result(
        self,
        *,
        status: str,
        skill_path: Path,
        started_at: str,
        ended_at: str,
        duration_seconds: float,
        return_code: int | None,
        stdout: str,
        stderr: str,
        output: Any | None,
        error: str | None,
        timed_out: bool,
        parameters: dict[str, Any] | None,
        output_parse_error: str | None,
    ) -> dict[str, Any]:
        return {
            "runner": "basic_runner",
            "status": status,
            "skill_path": str(skill_path),
            "execution_type": "python_subprocess",
            "output": output,
            "raw_stdout": stdout,
            "raw_stderr": stderr,
            "error": error,
            "metadata": {
                "schema_path": str(self.schema_path),
                "schema_loaded": self.schema is not None,
                "parameters_provided": parameters is not None,
                "timeout_seconds": self.timeout_seconds,
                "timed_out": timed_out,
                "return_code": return_code,
                "started_at": started_at,
                "ended_at": ended_at,
                "duration_seconds": round(duration_seconds, 6),
                "output_parse_error": output_parse_error,
            },
        }

    def _utc_now(self) -> str:
        return datetime.now(UTC).isoformat().replace("+00:00", "Z")

    def _coerce_output(self, output: str | bytes | None) -> str:
        if output is None:
            return ""
        if isinstance(output, bytes):
            return output.decode("utf-8", errors="replace")
        return output


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Run a skill with the basic runner.")
    parser.add_argument("skill_path", help="Path to the Python skill script.")
    parser.add_argument("--schema-path", default=None, help="Path to skill_schema.json.")
    parser.add_argument("--timeout", type=int, default=10, help="Execution timeout in seconds.")
    parser.add_argument("--input", default=None, help="JSON object passed to the skill as parameters.")
    args = parser.parse_args()

    input_parameters = json.loads(args.input) if args.input else None
    runner = BasicRunner(schema_path=args.schema_path, timeout_seconds=args.timeout)
    print(json.dumps(runner.run(args.skill_path, parameters=input_parameters), ensure_ascii=False, indent=2))
