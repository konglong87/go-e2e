#!/usr/bin/env python3
"""Offline checks for the real-provider memory gate; no settings or API access."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
import yaml

SPEC = importlib.util.spec_from_file_location(
    "memory_gate", Path(__file__).with_name("memory-release-acceptance.py"))
GATE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GATE)


class MemoryGateTests(unittest.TestCase):
    def test_context_includes_runtime_messages_and_system_blocks(self):
        context = GATE.request_context({"request": {
            "system": "SYSTEM",
            "system_blocks": [{"text": "BLOCK"}],
            "messages": [{"content": [{"text": "BODY_ONLY_MEMORY"}]}],
        }})
        for value in ("SYSTEM", "BLOCK", "BODY_ONLY_MEMORY"):
            self.assertIn(value, context)

    def test_saved_memory_requires_body_and_valid_metadata(self):
        code = "SYNTHETIC_CONFIRMATION"
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            index = root / "MEMORY.md"
            memory = root / "deploy.md"
            GATE.write_private(index, "- [deployment](deploy.md)\n")
            valid = "---\nname: deploy\ndescription: >-\n  deployment verification\n" \
                    "metadata:\n  type: project\n---\n" + code
            GATE.write_private(memory, valid)
            self.assertEqual(GATE.validate_saved_memory(root, code), memory.resolve())
            for invalid in (valid.replace("name: deploy", "name: []"),
                            valid.replace("type: project", "type: unknown"),
                            valid.replace("name: deploy", "name: " + code),
                            valid.replace("name: deploy", "name: [invalid")):
                with self.subTest(content=invalid):
                    GATE.write_private(memory, invalid)
                    with self.assertRaises((AssertionError, yaml.YAMLError)):
                        GATE.validate_saved_memory(root, code)
            GATE.write_private(memory, valid)
            GATE.write_private(index, "- [deployment](deploy.md)\n" + code)
            with self.assertRaises(AssertionError):
                GATE.validate_saved_memory(root, code)

    def test_index_cannot_escape_root(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "memory"
            GATE.write_private(root / "MEMORY.md", "- [private](../private.md)\n")
            with self.assertRaisesRegex(AssertionError, "escapes"):
                GATE.validate_saved_memory(root, "SYNTHETIC")

    def test_snapshot_detects_content_and_file_set_changes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            GATE.write_private(root / "MEMORY.md", "index")
            before = GATE.memory_snapshot(root)
            GATE.write_private(root / "MEMORY.md", "updated")
            self.assertNotEqual(before, GATE.memory_snapshot(root))
            GATE.write_private(root / "MEMORY.md", "index")
            self.assertEqual(before, GATE.memory_snapshot(root))
            GATE.write_private(root / "extra.md", "extra")
            self.assertNotEqual(before, GATE.memory_snapshot(root))


if __name__ == "__main__":
    unittest.main()
