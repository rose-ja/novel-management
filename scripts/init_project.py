#!/usr/bin/env python3
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
setup = ROOT / "project" / "PROJECT-SETUP.md"
readme = ROOT / "README.md"

def main():
    name = input("Project name: ").strip()
    if not name:
        raise SystemExit("Project name is required.")
    slug = re.sub(r"[^a-z0-9]+", "-", name.lower()).strip("-")
    print(f"\nProject name: {name}")
    print(f"Suggested slug: {slug}")
    print("\nNext steps:")
    print(f"1. Edit {setup.relative_to(ROOT)}")
    print("2. Fill in actual stack, commands, constraints and acceptance owner.")
    print("3. Review project/constitution.md and project/AGENTS.md.")
    print("4. Do not run business implementation until P0-P6 are approved.")
    print("\nThis script intentionally does not install dependencies or overwrite files.")

if __name__ == "__main__":
    main()
