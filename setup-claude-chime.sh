#!/usr/bin/env bash
set -e

CLAUDE_DIR="$HOME/.claude"
HOOK_DIR="$CLAUDE_DIR/hooks"
SETTINGS_FILE="$CLAUDE_DIR/settings.json"
HOOK_FILE="$HOOK_DIR/chime.sh"
BACKUP_FILE="$SETTINGS_FILE.bak.$(date +%s)"

mkdir -p "$HOOK_DIR"
mkdir -p "$CLAUDE_DIR"

############################
# 1) Create chime script
############################
cat > "$HOOK_FILE" <<'EOF'
#!/usr/bin/env bash

if command -v afplay >/dev/null 2>&1; then
  afplay /System/Library/Sounds/Glass.aiff
  exit 0
fi

if command -v paplay >/dev/null 2>&1; then
  paplay /usr/share/sounds/freedesktop/stereo/complete.oga
  exit 0
fi

if command -v aplay >/dev/null 2>&1; then
  aplay /usr/share/sounds/alsa/Front_Center.wav
  exit 0
fi

if grep -qi microsoft /proc/version 2>/dev/null; then
  powershell.exe -c "(New-Object Media.SoundPlayer 'C:\\Windows\\Media\\notify.wav').PlaySync();" >/dev/null 2>&1
  exit 0
fi

exit 0
EOF

chmod +x "$HOOK_FILE"

############################
# 2) Backup settings.json
############################
if [[ -f "$SETTINGS_FILE" ]]; then
  cp "$SETTINGS_FILE" "$BACKUP_FILE"
  echo "📦 Backup created: $BACKUP_FILE"
else
  echo "{}" > "$SETTINGS_FILE"
fi

############################
# 3) Merge Stop hook safely
############################
jq --arg hook_cmd "bash $HOOK_FILE" '
  .hooks = (.hooks // {}) |
  .hooks.Stop = [
    {
      "hooks": [
        {
          "type": "command",
          "command": $hook_cmd
        }
      ]
    }
  ]
' "$SETTINGS_FILE" > "$SETTINGS_FILE.tmp"

mv "$SETTINGS_FILE.tmp" "$SETTINGS_FILE"

############################
# 4) Done
############################
echo "✅ Claude Code chime hook installed safely"
echo "🔔 Existing settings preserved"
echo
echo "Test sound with:"
echo "  bash $HOOK_FILE"
