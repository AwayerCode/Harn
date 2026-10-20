#!/usr/bin/env python3
"""Compatibility API and CLI for shared design checks."""
import importlib.util
import sys
from pathlib import Path
source = Path(__file__).resolve().parents[2] / 'scripts/check_design.py'
sys.path.insert(0, str(source.parent))
spec = importlib.util.spec_from_file_location('swe_shared_design', source)
shared = importlib.util.module_from_spec(spec)
spec.loader.exec_module(shared)
check = shared.check
main = shared.main
if __name__ == '__main__':
    sys.exit(main())
