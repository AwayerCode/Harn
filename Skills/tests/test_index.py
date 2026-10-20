#!/usr/bin/env python3
import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
INDEX = ROOT / "Skills" / "index.json"
FRONT_MATTER = re.compile(r"\A---\n(.*?)\n---\n", re.S)


def load_index():
    return json.loads(INDEX.read_text(encoding="utf-8"))


def read_name(skill_md):
    match = FRONT_MATTER.match(skill_md.read_text(encoding="utf-8"))
    if not match:
        return None
    for line in match.group(1).splitlines():
        if line.startswith("name:"):
            return line.split(":", 1)[1].strip().strip('"')
    return None


class IndexTest(unittest.TestCase):
    def test_schema_version(self):
        self.assertEqual(load_index().get("schema_version"), 1)

    def test_entries_are_unique_and_complete(self):
        names = set()
        paths = set()
        for skill in load_index()["skills"]:
            self.assertTrue(skill.get("name"), skill)
            self.assertTrue(skill.get("summary"), skill)
            self.assertIsInstance(skill.get("tags"), list, skill)
            self.assertIn(skill["name"], skill["path"], skill)
            self.assertNotIn(skill["name"], names, f"duplicate name {skill['name']}")
            self.assertNotIn(skill["path"], paths, f"duplicate path {skill['path']}")
            names.add(skill["name"])
            paths.add(skill["path"])

    def test_paths_resolve_to_skills(self):
        for skill in load_index()["skills"]:
            directory = ROOT / skill["path"]
            skill_md = directory / "SKILL.md"
            self.assertTrue(skill_md.is_file(), skill_md)
            self.assertEqual(read_name(skill_md), skill["name"], skill_md)
            self.assertTrue((directory / "agents" / "openai.yaml").is_file(), directory)

    def test_every_skill_is_indexed(self):
        indexed = {skill["path"] for skill in load_index()["skills"]}
        for skill_md in (ROOT / "Skills").glob("*/SKILL.md"):
            relative = skill_md.parent.relative_to(ROOT).as_posix()
            if skill_md.parent.name.startswith("_"):
                continue
            self.assertIn(relative, indexed, f"{relative} is not in Skills/index.json")

    def test_template_is_not_indexed(self):
        indexed = {skill["path"] for skill in load_index()["skills"]}
        self.assertNotIn("Skills/_template", indexed)


if __name__ == "__main__":
    unittest.main(verbosity=2)
