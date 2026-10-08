"""Ordinary CLI with no CLI Bridger discovery support; configured by a sidecar."""

import argparse

parser = argparse.ArgumentParser(description="Print a message without a Bridger handshake.")
parser.add_argument("--message", default="Hello from a third-party CLI")
parser.add_argument("--uppercase", action="store_true")
args = parser.parse_args()
print(args.message.upper() if args.uppercase else args.message)
