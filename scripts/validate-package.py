#!/usr/bin/env python3
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1]
manifest = json.loads((root / ".codex-plugin" / "plugin.json").read_text())
assert manifest["name"] == "tekton"
assert manifest["version"] == "0.1.0"
assert manifest["mcpServers"] == "./.mcp.json"
assert manifest["skills"] == "./skills/"
assert len(manifest["interface"]["defaultPrompt"]) == 3
skill_paths = sorted((root / "skills").glob("*/SKILL.md"))
assert len(skill_paths) == 8
for skill_file in skill_paths:
    path = skill_file.parent
    assert (path / "SKILL.md").is_file(), path
    prompt = (path / "agents" / "openai.yaml").read_text().splitlines()
    defaults = [line for line in prompt if line.strip().startswith("default_prompt: ")]
    assert len(defaults) == 1 and defaults[0].count("\n") == 0, path
for required in ["README.md", "UNLICENSE", "CONTEXT.md", ".mcp.json", "scripts/launch-tekton-mcp.sh"]:
    assert (root / required).is_file(), required
print("plugin package structure valid")
