"""Cooperating Python CLI: discovery JSON and a harmless terminal progress demo."""

import argparse
import json
import os
import sys
import time


DESCRIPTOR = {
    "version": "1",
    "name": "Python CLI Bridger Demo",
    "description": "A Python script with a live progress bar; no files are modified.",
    "root": {
        "id": "root",
        "name": "",
        "description": "Choose render to demonstrate live terminal output from Python.",
        "parameters": [{"id": "demoRoot", "name": "Demo root directory",
                        "description": "Override BRIDGER_DEMO_ROOT for this child process; the demo prints it without accessing files.",
                        "env": "BRIDGER_DEMO_ROOT", "type": "path", "pathKind": "directory", "default": "."}],
        "commands": [{
            "id": "render",
            "name": "render",
            "description": "Render a terminal progress bar",
            "parameters": [
                {"id": "steps", "name": "Steps", "flag": "--steps", "type": "int",
                 "description": "Number of progress updates, with a 0.05-second pause after each update.",
                 "required": True, "default": 20, "examples": [10, 50],
                 "limits": {"min": 1, "max": 100}},
                {"id": "color", "name": "Colored output", "flag": "--color",
                 "description": "Display the progress bar in cyan using ANSI terminal color codes.",
                 "type": "bool", "default": True},
                {"id": "label", "name": "Progress label", "flag": "--label",
                 "description": "Text printed above the progress bar; editable here when colored output is enabled.",
                 "type": "string", "default": "Hello from Python",
                 "examples": ["Processing files", "Building assets"],
                 "limits": {"minLength": 1, "maxLength": 80},
                 "dependsOn": {"id": "color", "value": True}},
            ],
        }],
    },
}


def main():
    if sys.argv[1:] == ["--cli-bridger-describe"]:
        print(json.dumps(DESCRIPTOR), flush=True)
        return

    parser = argparse.ArgumentParser(description=DESCRIPTOR["description"])
    commands = parser.add_subparsers(dest="command", required=True)
    render = commands.add_parser("render")
    render.add_argument("--steps", type=int, default=20, choices=range(1, 101))
    render.add_argument("--label", default="Hello from Python")
    render.add_argument("--color", action="store_true")
    args = parser.parse_args()
    if not 1 <= len(args.label) <= 80:
        parser.error("label must contain 1 to 80 characters")
    print(args.label, flush=True)
    print("BRIDGER_DEMO_ROOT:", os.environ.get("BRIDGER_DEMO_ROOT", ""), flush=True)
    prefix, suffix = ("\033[36m", "\033[0m") if args.color else ("", "")
    for step in range(args.steps + 1):
        filled = 30 * step // args.steps
        bar = "=" * filled + " " * (30 - filled)
        print(f"\r{prefix}[{bar}] {100 * step // args.steps:3d}%{suffix}", end="", flush=True)
        time.sleep(0.05)
    print("\nDone. No files were modified.", flush=True)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\nStopped.", file=sys.stderr)
        sys.exit(130)
