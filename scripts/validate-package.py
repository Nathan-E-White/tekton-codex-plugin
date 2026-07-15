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

integration_contract = {
    "scripts/kind-smoke.sh": [
        "pipeline/releases/download/v1.14.0/release.yaml",
        "triggers/releases/download/v0.36.0/release.yaml",
        "chains/releases/download/v0.28.0/release.yaml",
        "results/releases/download/v0.19.0/release.yaml",
        "pipelines-as-code/releases/download/v0.49.0/release.k8s.yaml",
    ],
    "scripts/github-app-smoke.sh": [
        "GITHUB_APP_ID",
        "GITHUB_APP_INSTALLATION_ID",
        "GITHUB_APP_PRIVATE_KEY",
        "GITHUB_WEBHOOK_SECRET",
        "/test",
        "/retest",
        "/cancel",
    ],
}
for relative, markers in integration_contract.items():
    body = (root / relative).read_text()
    for marker in markers:
        assert marker in body, f"{relative} must cover {marker}"
print("plugin package structure valid")
