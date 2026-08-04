#!/usr/bin/env bash
set -euo pipefail
ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
PINNED='v0.15.2'; EXPECTED='0.15.2'; SOURCE="git+https://github.com/github/spec-kit.git@${PINNED}"
command -v git >/dev/null || { echo 'git is required.' >&2; exit 2; }
if ! command -v uv >/dev/null; then command -v curl >/dev/null || { echo 'curl is required to install uv.' >&2; exit 2; }; curl -LsSf https://astral.sh/uv/install.sh | sh; export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$PATH"; fi
uv tool install specify-cli --force --from "$SOURCE"; export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$PATH"
command -v specify >/dev/null || { echo 'specify is not visible on PATH.' >&2; exit 2; }
specify version | grep -F "$EXPECTED" >/dev/null || { echo "Expected Spec Kit $EXPECTED" >&2; exit 2; }
cd "$ROOT"
if [[ ! -f .specify/integration.json ]]; then specify init --here --force --integration codex --script sh --ignore-agent-tools; elif [[ ! -f .agents/skills/speckit-specify/SKILL.md ]]; then specify integration install codex --script sh --force; specify integration use codex --force; fi
mkdir -p .specify/templates/overrides .agents/skills; cp -a speckit/overrides/. .specify/templates/overrides/; cp -a speckit/ylven-skills/. .agents/skills/
specify integration status
uv run --python 3.12 --with-requirements requirements-tools.txt python scripts/12_REGENERATE_CONTRACTS.py
uv run --python 3.12 --with-requirements requirements-tools.txt python scripts/07_VALIDATE_CONTRACTS.py
uv run --python 3.12 --with-requirements requirements-tools.txt python scripts/38_VALIDATE_STATE_CONTINUITY.py
printf 'Spec Kit %s installed. Heavy build tools remain on GitHub Actions and the SSH-connected server.\n' "$PINNED"
