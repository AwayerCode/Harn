#!/usr/bin/env python3
"""Compatibility entry; schema and validation logic are owned by SWE standards."""
import importlib.util
import sys
from pathlib import Path

source = Path(__file__).resolve().parents[2] / 'scripts/validate_documents.py'
if not source.is_file():
    raise RuntimeError('缺少 SWE 共享校验器；请安装完整 SWE 包')
spec = importlib.util.spec_from_file_location('swe_document_validator', source)
shared = importlib.util.module_from_spec(spec)
spec.loader.exec_module(shared)
cells = shared.cells
tables = shared.tables
outside_code = shared.outside_code
validate = shared.validate
resolve_contract = shared.resolve_contract
load_contract = shared.load_contract

if __name__ == '__main__':
    sys.exit(shared.main(Path(__file__).resolve().parents[1]))
