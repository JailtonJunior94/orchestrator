package uninstall

const installClaudeSettingsTemplate = `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit|apply_patch",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/validate-preload.sh"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit|apply_patch",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/validate-governance.sh"
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "matcher": "task-executor",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/subagent-stop-wrapper.sh"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/validate-session-end.sh"
          }
        ]
      }
    ]
  }
}
`

const legacyClaudeSettingsTemplate = `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/validate-preload.sh"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/validate-governance.sh"
          }
        ]
      }
    ]
  }
}
`
