#!/usr/bin/env bash
set -euo pipefail

# gwx installer — installs CLI binary + Claude Code skill
# Usage: ./install.sh [--global|--project]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCOPE="${1:---global}"

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║         gwx — Installer                  ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# --- Step 1: Build & install CLI ---
echo "→ Building gwx CLI..."
cd "$SCRIPT_DIR"

if command -v go &>/dev/null; then
    go install -ldflags "-s -w" ./cmd/gwx/
    GWX_BIN="$(go env GOPATH)/bin/gwx"
    echo "  ✓ Installed to $GWX_BIN"
else
    echo "  ✗ Go not found. Building binary to ./build/gwx instead."
    mkdir -p build
    # Try downloading prebuilt or instruct manual install
    echo "  Please install Go from https://go.dev/dl/ and re-run."
    exit 1
fi

# Verify
if ! command -v gwx &>/dev/null; then
    echo "  ⚠ gwx not in PATH. Add $(go env GOPATH)/bin to your PATH:"
    echo "    export PATH=\"\$PATH:$(go env GOPATH)/bin\""
fi

# --- Step 2: Install Claude Code skill ---
echo ""
echo "→ Installing Claude Code skill..."

if [ "$SCOPE" = "--project" ]; then
    # Project-level: .claude/commands/ in current working directory's project
    SKILL_DIR=".claude/commands"
    AGENT_DIR=".claude/agents"
    echo "  Mode: project-level ($SKILL_DIR)"
else
    # Global: ~/.claude/commands/
    SKILL_DIR="$HOME/.claude/commands"
    AGENT_DIR="$HOME/.claude/agents"
    echo "  Mode: global ($SKILL_DIR)"
fi

mkdir -p "$SKILL_DIR"
mkdir -p "$AGENT_DIR"

# Copy main skill
cp "$SCRIPT_DIR/skill/google-workspace.md" "$SKILL_DIR/google-workspace.md"
echo "  ✓ Skill: google-workspace.md"

# Copy agents
for agent in "$SCRIPT_DIR"/skill/agents/*.md; do
    name="$(basename "$agent")"
    cp "$agent" "$AGENT_DIR/$name"
    echo "  ✓ Agent: $name"
done

# Copy recipes as skills (they're invocable as slash commands)
for recipe in "$SCRIPT_DIR"/skill/recipes/*.md; do
    name="$(basename "$recipe")"
    cp "$recipe" "$SKILL_DIR/gwx-${name}"
    echo "  ✓ Recipe: gwx-${name}"
done

# --- Step 3: Install YAML skills ---
echo ""
echo "→ Installing YAML skills..."
GWX_SKILLS_DIR="$HOME/.config/gwx/skills"
mkdir -p "$GWX_SKILLS_DIR"

for skill_file in "$SCRIPT_DIR"/skills/*.yaml "$SCRIPT_DIR"/skills/*.yml; do
    [ -f "$skill_file" ] || continue
    name="$(basename "$skill_file")"
    cp "$skill_file" "$GWX_SKILLS_DIR/$name"
    echo "  ✓ Skill: $name"
done

# --- Step 4: Verify ---
echo ""
echo "→ Verifying..."

if command -v gwx &>/dev/null; then
    VERSION=$(gwx version --format plain 2>/dev/null || echo "unknown")
    echo "  ✓ gwx CLI: $VERSION"
else
    echo "  ⚠ gwx not in PATH (install succeeded but PATH needs updating)"
fi

echo "  ✓ Skills installed to $SKILL_DIR"
echo "  ✓ Agents installed to $AGENT_DIR"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✓ Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'gwx onboard' to set up Google credentials"
echo "  2. In Claude Code, trigger with: 'check my email' or '看一下行事曆'"
echo ""
