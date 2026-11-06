#!/usr/bin/env python3
from pathlib import Path
import argparse
import re
import shutil
from datetime import date

ROOT = Path(__file__).resolve().parents[1]
TEMPLATE = ROOT / "changes" / "_template"
TARGET_ROOT = ROOT / "changes"

def main():
    parser = argparse.ArgumentParser(description="Create a change folder from template.")
    parser.add_argument("change_id", help="e.g. CHG-001")
    parser.add_argument("name", help="short-kebab-case name")
    args = parser.parse_args()

    if not re.fullmatch(r"CHG-\d{3,}", args.change_id):
        raise SystemExit("change_id must look like CHG-001")
    if not re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", args.name):
        raise SystemExit("name must be lowercase kebab-case")

    target = TARGET_ROOT / f"{args.change_id}-{args.name}"
    if target.exists():
        raise SystemExit(f"Already exists: {target}")

    shutil.copytree(TEMPLATE, target)
    for path in target.glob("*.md"):
        content = path.read_text(encoding="utf-8")
        content = content.replace("{{CHANGE_ID}}", args.change_id)
        content = content.replace("{{DATE}}", date.today().isoformat())
        path.write_text(content, encoding="utf-8")

    print(f"Created: {target.relative_to(ROOT)}")
    print("Next: fill proposal.md, then run F1 task breakdown.")

if __name__ == "__main__":
    main()
