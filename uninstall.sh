#!/usr/bin/env bash
set -euo pipefail

echo "→ Removing gwx..."

# Remove binary
GWX_BIN="$(command -v gwx 2>/dev/null || echo "")"
if [ -n "$GWX_BIN" ]; then
    rm -f "$GWX_BIN"
    echo "  ✓ Removed binary: $GWX_BIN"
fi

# Remove global skills
for f in "$HOME/.claude/commands/google-workspace.md" "$HOME/.claude/commands/gwx-"*.md; do
    [ -f "$f" ] && rm -f "$f" && echo "  ✓ Removed: $f"
done

# Remove global agents
for f in "$HOME/.claude/agents/gmail-agent.md" \
         "$HOME/.claude/agents/calendar-agent.md" \
         "$HOME/.claude/agents/drive-agent.md" \
         "$HOME/.claude/agents/workspace-router.md"; do
    [ -f "$f" ] && rm -f "$f" && echo "  ✓ Removed: $f"
done

# Remove keyring tokens (optional)
echo ""
echo "  Note: OAuth tokens in OS Keyring are NOT removed."
echo "  To remove them manually: gwx auth logout (before uninstall)"
echo ""
echo "✓ Uninstall complete."
