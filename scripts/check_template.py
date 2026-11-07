#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
required = [
    "README.md",
    "workflow/00-operating-manual.md",
    "workflow/01-project-bootstrap.md",
    "workflow/02-change-cycle.md",
    "project/PROJECT-SETUP.md",
    "project/constitution.md",
    "project/AGENTS.md",
    "changes/_template/proposal.md",
    "changes/_template/design.md",
    "changes/_template/tasks.md",
    "changes/_template/checklist.md",
    "changes/_template/verification.md",
    "changes/_template/acceptance.md",
]
missing = [p for p in required if not (ROOT / p).is_file()]
if missing:
    print("Missing required template files:")
    for item in missing:
        print(f"- {item}")
    sys.exit(1)
print(f"Template structure OK ({len(required)} required files checked).")
