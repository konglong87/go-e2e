#!/usr/bin/env python3
"""Real-provider memory release gate. Requires PyYAML; never uses real memories."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys
import time

PROVIDER = "sensenova-glm-5.2"
WRITE_TOOLS = "Read,Grep,Write,Edit"
NO_TOOLS = ""
TRUNCATION_MARKER = "[Memory truncated for prompt budget]"


def write_private(path, content):
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    path.write_text(content, encoding="utf-8")
    path.chmod(0o600)


def memory_snapshot(directory):
    return {str(p.relative_to(directory)): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in directory.rglob("*") if p.is_file()}


def request_context(record):
    request = record["request"]
    # Code context is carried in a runtime message, not necessarily system.
    return json.dumps({key: request.get(key) for key in ("system", "system_blocks", "messages")},
                      ensure_ascii=False)


def validate_saved_memory(directory, code):
    import yaml

    index = (directory / "MEMORY.md").read_text()
    assert code not in index, "answer leaked into the always-loaded index"
    links = re.findall(r"\[[^\]]+\]\(([^)]+\.md)\)", index)
    matches = []
    for link in links:
        path = (directory / link).resolve()
        assert directory.resolve() in path.parents, "index escapes the isolated memory root"
        if not path.is_file():
            continue
        content = path.read_text()
        if code not in content:
            continue
        assert content.startswith("---\n"), "missing generated YAML frontmatter"
        header, separator, _ = content[4:].partition("\n---")
        assert separator, "unterminated generated frontmatter"
        assert code not in header, "answer must be in the memory body, not metadata"
        metadata = yaml.safe_load(header)
        assert isinstance(metadata, dict), "frontmatter must be a mapping"
        assert isinstance(metadata.get("name"), str) and metadata["name"].strip()
        assert isinstance(metadata.get("description"), str) and metadata["description"].strip()
        assert metadata.get("metadata", {}).get("type") in {"project", "user", "feedback", "reference"}
        matches.append(path)
    assert len(matches) == 1, "expected one indexed memory containing the saved fact"
    return matches[0]


class Gate:
    def __init__(self, args):
        self.args = args
        self.root = Path(args.out_dir).resolve()
        self.root.mkdir(mode=0o700, parents=True, exist_ok=False)
        self.workspace_a = self.root / "project-a"
        self.workspace_b = self.root / "project-b"
        self.home = self.root / "home"
        self.config = self.home / "config"
        for path in (self.workspace_a, self.workspace_b, self.config, self.root / "tmp"):
            path.mkdir(parents=True, mode=0o700)
        self.reject_ambient_guidance()
        source = Path(args.settings).read_bytes()
        self.source_hash = hashlib.sha256(source).hexdigest()
        settings = json.loads(source)
        selected = next(p for p in settings["fallback"]["providers"] if p["name"] == args.provider)
        assert selected.get("apiKey"), "selected provider has no inline API key"
        self.model = selected["model"]
        self.api_key = selected["apiKey"]
        write_private(self.config / "settings.json", json.dumps({
            "model": self.model,
            "fallback": {"providers": [selected]},
        }))
        self.memory = self.config / "projects" / str(self.workspace_a).replace("/", "-").replace(":", "").replace(" ", "-") / "memory"
        self.memory.mkdir(parents=True, mode=0o700)
        self.code = "RELEASE-" + secrets.token_hex(8).upper()
        self.results = []

    def reject_ambient_guidance(self):
        if Path("/etc/claude-code/CLAUDE.md").exists():
            raise RuntimeError("managed guidance is present; use an isolated host")
        for ancestor in self.workspace_a.parents:
            for name in ("golang-cc.md", "golang-claude-code.md", "CLAUDE.md", "AGENTS.md",
                         "CLAUDE.local.md", ".claude/CLAUDE.md", ".claude/rules",
                         ".claude/workflows", "SKILL.md", "WORKFLOW.md", "CONTRACT.md",
                         "references/README.md", "references/WORKFLOW.md", "docs/WORKFLOW.md"):
                if (ancestor / name).exists():
                    raise RuntimeError("out-dir inherits project guidance; choose a directory outside the checkout")

    def env(self, extra=None):
        result = {
            "PATH": os.environ["PATH"], "HOME": str(self.home), "USERPROFILE": str(self.home),
            "SHELL": os.environ.get("SHELL", "/bin/sh"), "LANG": "C", "LC_ALL": "C",
            "TMPDIR": str(self.root / "tmp"), "GOLANG_CC_CONFIG_DIR": str(self.config),
            "CLAUDE_CONFIG_DIR": str(self.config), "GOLANG_CC_PROMPT_PROFILE": "claude-compatible",
            "GOLANG_CC_DUMP_PROMPT_FULL": "true",
        }
        result.update(extra or {})
        return result

    def run(self, name, prompt, *, workspace=None, turns=2, tools=NO_TOOLS, extra=None, cli_args=()):
        dump = self.root / (name + ".prompt.jsonl")
        env = self.env(dict(extra or {}, GOLANG_CC_DUMP_PROMPT_JSON=str(dump)))
        command = [str(Path(self.args.binary).resolve()), "--cwd", str(workspace or self.workspace_a),
                   "--provider", self.args.provider, "--model", self.model,
                   "--tools", tools, "--max-turns", str(turns), "--max-tokens", "2048",
                   "--output-format", "json", *cli_args, "-p", prompt]
        started = time.monotonic()
        completed = subprocess.run(command, cwd=self.root, env=env, capture_output=True,
                                   text=True, timeout=self.args.timeout)
        write_private(self.root / (name + ".stdout.json"), completed.stdout)
        write_private(self.root / (name + ".stderr.log"), completed.stderr)
        assert completed.returncode == 0, f"{name}: CLI failed; inspect private evidence"
        result = json.loads(completed.stdout)
        records = [json.loads(line) for line in dump.read_text().splitlines()]
        assert records and all(record["model"] == self.model for record in records)
        assert self.api_key not in dump.read_text(), "credential appeared in a prompt dump"
        system = request_context(records[0])
        context = records[0].get("context_manifest", {}).get("code_context", {})
        self.results.append({
            "case": name, "model": result["model"], "turns": result["turns"],
            "tools": [trace["name"] for trace in result.get("tool_calls", [])],
            "usage": result.get("usage", {}), "request_count": len(records),
            "first_system_bytes": records[0]["system_bytes"],
            "first_request_context_bytes": len(system.encode()),
            "memory_prompt_bytes": context.get("prompt_bytes", 0),
            "memory_documents": context.get("documents", 0),
            "elapsed_seconds": round(time.monotonic() - started, 3),
        })
        return result, system, context

    def check_readonly(self, name, prompt, expected, *, workspace=None, tools=NO_TOOLS, turns=2,
                       cli_args=()):
        assert self.code not in prompt, "recall prompt contains the expected answer"
        before = memory_snapshot(self.memory)
        result, system, _ = self.run(name, prompt, workspace=workspace, tools=tools, turns=turns,
                                     cli_args=cli_args)
        assert (self.code in system) == expected, f"{name}: wrong first-request memory body"
        assert (self.code in result["response"]) == expected, f"{name}: unexpected answer"
        assert memory_snapshot(self.memory) == before, f"{name}: memory changed"
        assert not any(t["name"] in {"Write", "Edit", "MultiEdit"} for t in result.get("tool_calls", []))
        if not tools:
            assert not result.get("tool_calls"), f"{name}: cold recall must not rely on tool reads"

    def save_and_restart(self):
        self.run("write", "Remember this durable project convention: for deployment verification, "
                 f"the pre-release confirmation code is {self.code}. Save it as a project memory "
                 "with a descriptive name and description for deployment verification. "
                 "Keep the code only in the memory body, never in the index or frontmatter. "
                 "This is a lasting release policy, not an activity summary. Reply saved when persisted.",
                 turns=6, tools=WRITE_TOOLS)
        path = validate_saved_memory(self.memory, self.code)
        write_private(self.root / "saved-index.md", (self.memory / "MEMORY.md").read_text())
        write_private(self.root / "saved-memory.md", path.read_text())
        self.check_readonly("cold-recall", "For deployment verification, what is this project's "
                            "pre-release confirmation code? Answer only the code from context.", True)
        # The expected flag is explicit so a negative scenario cannot pass via a canned answer.
        return path

    def negative_cases(self, saved):
        prompt = "For deployment verification, what is this project's pre-release confirmation code? " \
                 "Answer from context only; say unknown if unavailable."
        self.check_readonly("other-project", prompt, False, workspace=self.workspace_b)
        self.check_readonly("chat-mode", prompt, False, cli_args=("--prompt-mode", "chat"))
        self.check_readonly("bare-mode", prompt, False, cli_args=("--bare",))
        self.check_readonly("ignore", "Ignore memory. For deployment verification, do not use stored "
                            "conventions. Reply only ignore-memory-ok.", False)
        original_index = (self.memory / "MEMORY.md").read_text()
        original_body = saved.read_text()
        try:
            write_private(self.memory / "MEMORY.md", "# Memory\nNo indexed entries.\n")
            self.check_readonly("unindexed", prompt, False)
            write_private(self.memory / "MEMORY.md", original_index)
            saved.unlink()
            self.check_readonly("missing-file", prompt, False)
        finally:
            write_private(saved, original_body)
            write_private(self.memory / "MEMORY.md", original_index)
        self.check_readonly("readonly-write-tools", prompt, True, tools=WRITE_TOOLS, turns=3)

    def budget_case(self):
        marker = "BUDGET_PRIVATE_TAIL"
        index = "".join(f"- [Budget {i}](budget-{i}.md)\n" for i in range(6))
        write_private(self.memory / "MEMORY.md", index)
        for i in range(6):
            write_private(self.memory / f"budget-{i}.md", "---\nname: budget verification\n"
                          "description: bytebudget regression\nmetadata:\n  type: project\n---\n" +
                          ("Budget opening guidance.\n" * 180) + marker)
        before = memory_snapshot(self.memory)
        _, system, context = self.run(
            "budget", "For budget verification and bytebudget regression, reply budget-ok.",
            extra={"GOLANG_CC_MEMORY_DOCUMENT_BUDGET_BYTES": "1024",
                   "GOLANG_CC_MEMORY_PROMPT_BUDGET_BYTES": "2048"})
        docs = context.get("document_summary", [])
        recalled = [d for d in docs if d.get("type") == "ClaudeCodeProjectMemoryRecall"]
        assert len(recalled) == 4, "recall file-count bound was not applied"
        assert any(d.get("budgeted") for d in recalled), "document budget never activated"
        assert recalled[0]["prompt_bytes"] > 800, "fixture did not consume the document budget"
        assert recalled[-1]["prompt_bytes"] <= 200 + len(recalled[-1]["path"].encode()), \
            "total budget did not reduce the last document to its read pointer"
        assert marker not in system and TRUNCATION_MARKER in system
        # Existing budgets retain one complete read-pointer per exhausted document.
        notice_overhead = sum(200 + len(d["path"].encode()) for d in docs)
        assert context["prompt_bytes"] <= 2048 + notice_overhead
        assert context["prompt_bytes"] == sum(d.get("prompt_bytes", 0) for d in docs)
        assert all(d.get("prompt_bytes", 0) <= max(1024, 200 + len(d["path"].encode())) for d in docs)
        assert memory_snapshot(self.memory) == before
        self.results[-1]["configured_content_budget"] = 2048
        self.results[-1]["read_pointer_overhead_bytes"] = max(0, context["prompt_bytes"] - 2048)

    def finish(self):
        assert hashlib.sha256(Path(self.args.settings).read_bytes()).hexdigest() == self.source_hash
        report = {"ok": True, "provider": self.args.provider, "model": self.model,
                  "global_settings_unchanged": True,
                  "binary_sha256": hashlib.sha256(Path(self.args.binary).read_bytes()).hexdigest(),
                  "cases": self.results}
        write_private(self.root / "summary.json", json.dumps(report, indent=2))
        print(json.dumps(report))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True)
    parser.add_argument("--out-dir", required=True, help="New persistent directory outside the checkout; never /tmp")
    parser.add_argument("--settings", default=str(Path.home() / ".golang-cc/settings.json"))
    parser.add_argument("--provider", default=PROVIDER)
    parser.add_argument("--timeout", type=int, default=240, help="Per-process timeout in seconds")
    args = parser.parse_args()
    output = Path(args.out_dir).resolve()
    if output == Path("/tmp") or Path("/tmp").resolve() in output.parents:
        parser.error("credentials and evidence must not be stored in /tmp")
    os.umask(0o077)
    gate = Gate(args)
    try:
        saved = gate.save_and_restart()
        gate.negative_cases(saved)
        gate.budget_case()
        gate.finish()
    except Exception as error:
        write_private(gate.root / "summary.json", json.dumps({
            "ok": False, "error_type": type(error).__name__, "error": str(error),
            "completed_cases": gate.results,
        }, indent=2))
        raise


if __name__ == "__main__":
    main()
