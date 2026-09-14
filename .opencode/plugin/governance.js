import { existsSync, writeFileSync } from "node:fs"
import { spawnSync } from "node:child_process"
import { join } from "node:path"

const SENTINEL_ENV_VAR = "AISPEC_OPENCODE_GOVERNANCE_SENTINEL"
const SESSION_IDLE_SENTINEL_ENV_VAR = "AISPEC_OPENCODE_SESSION_IDLE_SENTINEL"
const ORCHESTRATED_ENV_VAR = "AISPEC_OPENCODE_ORCHESTRATED"
const CANONICAL_PRE_TOOL_SCRIPT = ".agents/hooks/validate-preload.sh"
const CANONICAL_POST_TOOL_SCRIPT = ".agents/hooks/validate-governance.sh"
const CANONICAL_SESSION_END_SCRIPT = ".agents/scripts/validate-session-end.sh"
const MUTATING_TOOLS = new Set(["bash", "write", "edit", "multiedit", "patch", "apply_patch"])
const READ_ONLY_TOOLS = new Set(["read", "glob", "grep", "list", "ls", "webfetch", "websearch", "todoread", "todowrite", "task"])
const COMMAND_ARG_KEYS = ["command", "cmd", "script"]
const SOURCE_FILE_PATTERN = /\.(go|ts|tsx|js|jsx|mjs|cjs|py|cs|csproj)$/
const COMMAND_TOKEN_SEPARATORS = /[\s;|&()<>]+/
const VALIDATOR_TIMEOUT_MS = Number(process.env.AISPEC_OPENCODE_VALIDATOR_TIMEOUT_MS) || 8000

const sentinelPath = process.env[SENTINEL_ENV_VAR]
if (sentinelPath) {
  writeFileSync(sentinelPath, "")
}

function writeSessionIdleStatus(status) {
  const path = process.env[SESSION_IDLE_SENTINEL_ENV_VAR]
  if (path) {
    writeFileSync(path, status)
  }
}

const memo = new Map()
let validatorExistsCache = null
let sessionEndValidatorExistsCache = null
let interactiveWarnedOnce = false
let sessionEndInteractiveWarnedOnce = false
const unrecognizedToolWarned = new Set()

function isOrchestrated() {
  return process.env[ORCHESTRATED_ENV_VAR] === "1"
}

function unknownToolDenial(tool) {
  return "tool \"" + tool + "\" is neither in the known mutating-tools set (" +
    Array.from(MUTATING_TOOLS).join(", ") +
    ") nor in the read-only allowlist (" +
    Array.from(READ_ONLY_TOOLS).join(", ") +
    ") — an unclassified tool is treated as denial under orchestration, never approval; classify it in governance.js before retrying"
}

function warnUnrecognizedToolOnce(tool) {
  if (unrecognizedToolWarned.has(tool)) {
    return
  }
  unrecognizedToolWarned.add(tool)
  console.warn("aispec governance: tool \"" + tool + "\" is not in the known mutating-tools set (" +
    Array.from(MUTATING_TOOLS).join(", ") +
    ") — proceeding without validation; add it to MUTATING_TOOLS in governance.js if this tool can mutate the repository")
}

function extractTouchedFiles(args) {
  if (typeof args === "string") {
    const match = args.match(/^\*\*\* (?:Add|Update|Delete) File: (.+)$/m)
    return match ? [match[1]] : []
  }
  if (!args || typeof args !== "object") {
    return []
  }
  for (const key of ["patch", "patchText"]) {
    if (typeof args[key] === "string") {
      return extractTouchedFiles(args[key])
    }
  }
  const files = []
  for (const key of ["filePath", "file_path", "path", "file"]) {
    const value = args[key]
    if (typeof value === "string" && value.length > 0) {
      files.push(value)
    }
  }
  return files
}

function extractCommandText(args) {
  if (typeof args === "string") {
    return args
  }
  if (!args || typeof args !== "object") {
    return null
  }
  for (const key of COMMAND_ARG_KEYS) {
    const value = args[key]
    if (typeof value === "string" && value.trim().length > 0) {
      return value
    }
    if (Array.isArray(value) && value.length > 0) {
      return value.map(String).join(" ")
    }
  }
  return null
}

function extractCommandTargets(command) {
  const targets = []
  for (const rawToken of command.split(COMMAND_TOKEN_SEPARATORS)) {
    const token = rawToken.replace(/^["'`]+/, "").replace(/["'`]+$/, "")
    if (token.length === 0 || token.startsWith("-")) {
      continue
    }
    if (SOURCE_FILE_PATTERN.test(token) && !targets.includes(token)) {
      targets.push(token)
    }
  }
  return targets
}

function resolveValidationTargets(tool, args) {
  const files = extractTouchedFiles(args)
  if (files.length > 0) {
    return { targets: files }
  }
  const command = extractCommandText(args)
  if (command === null) {
    return {
      targets: [],
      denial: "no mutation target could be extracted from the arguments of tool \"" + tool +
        "\" — absence of target is treated as denial, never approval",
    }
  }
  return { targets: extractCommandTargets(command), command }
}

function hookPayload(filePath) {
  return JSON.stringify({ tool_input: { file_path: filePath } })
}

function commandPayload(command) {
  return JSON.stringify({ tool_input: { command } })
}

function memoKey(tool, files) {
  return tool + "::" + files.slice().sort().join(",")
}

function commandMemoKey(tool, command) {
  return tool + "::command::" + command
}

function validatorExists(scriptPath) {
  if (validatorExistsCache === null) {
    validatorExistsCache = existsSync(scriptPath)
  }
  return validatorExistsCache
}

function sessionEndValidatorExists(scriptPath) {
  if (sessionEndValidatorExistsCache === null) {
    sessionEndValidatorExistsCache = existsSync(scriptPath)
  }
  return sessionEndValidatorExistsCache
}

function correctiveMessage(tool, reason) {
  return "GOVERNANCE BLOCKED for tool \"" + tool + "\": " + reason +
    "\nFix the reported condition and retry the same tool call — do not attempt to route around this gate."
}

function runCanonicalValidator(directory, payloads) {
  const scriptPath = join(directory, CANONICAL_PRE_TOOL_SCRIPT)
  if (!validatorExists(scriptPath)) {
    const orchestrated = isOrchestrated()
    if (orchestrated) {
      return { blocked: true, reason: "canonical validator " + CANONICAL_PRE_TOOL_SCRIPT + " is missing — run 'ai-spec-harness install .' before retrying" }
    }
    if (!interactiveWarnedOnce) {
      interactiveWarnedOnce = true
      console.warn("aispec governance: canonical validator missing at " + scriptPath + " — proceeding without gate (interactive mode, warned once)")
    }
    return { blocked: false }
  }
  for (const payload of payloads) {
    const result = spawnSync("bash", [scriptPath], {
      cwd: directory,
      timeout: VALIDATOR_TIMEOUT_MS,
      input: payload,
      encoding: "utf8",
    })
    if (result.error && result.error.code === "ETIMEDOUT") {
      return { blocked: true, reason: "validator timed out after " + VALIDATOR_TIMEOUT_MS + "ms — timeout is treated as denial, never approval" }
    }
    if (result.status !== 0) {
      const detail = (result.stderr || "").trim()
      return { blocked: true, reason: detail.length > 0 ? detail : "canonical validator rejected this tool call" }
    }
  }
  return { blocked: false }
}

function runSessionEndValidator(directory) {
  const scriptPath = join(directory, CANONICAL_SESSION_END_SCRIPT)
  if (!sessionEndValidatorExists(scriptPath)) {
    const orchestrated = isOrchestrated()
    if (orchestrated) {
      return { blocked: true, reason: "canonical validator " + CANONICAL_SESSION_END_SCRIPT + " is missing — run 'ai-spec-harness install .' before retrying" }
    }
    if (!sessionEndInteractiveWarnedOnce) {
      sessionEndInteractiveWarnedOnce = true
      console.warn("aispec governance: canonical session-end validator missing at " + scriptPath + " — proceeding without gate (interactive mode, warned once)")
    }
    return { blocked: false }
  }
  const result = spawnSync("bash", [scriptPath], {
    cwd: directory,
    timeout: VALIDATOR_TIMEOUT_MS,
    input: "",
    encoding: "utf8",
  })
  if (result.error && result.error.code === "ETIMEDOUT") {
    return { blocked: true, reason: "session-end validator timed out after " + VALIDATOR_TIMEOUT_MS + "ms — timeout is treated as denial, never approval" }
  }
  if (result.status !== 0) {
    const detail = (result.stderr || "").trim()
    return { blocked: true, reason: detail.length > 0 ? detail : "canonical session-end validator rejected session end" }
  }
  return { blocked: false }
}

function postToolPayloads(tool, args) {
  const resolved = resolveValidationTargets(tool, args)
  if (resolved.targets.length > 0) {
    return resolved.targets.map(hookPayload)
  }
  if (typeof resolved.command === "string") {
    return [commandPayload(resolved.command)]
  }
  return []
}

function observePostTool(directory, tool, args) {
  const scriptPath = join(directory, CANONICAL_POST_TOOL_SCRIPT)
  if (!existsSync(scriptPath)) {
    console.warn("aispec governance: canonical post-tool validator missing at " + scriptPath)
    return
  }
  const payloads = postToolPayloads(tool, args)
  if (payloads.length === 0) {
    console.warn("aispec governance: no post-tool target extracted for tool \"" + tool +
      "\" — the canonical validator was not given anything to observe")
    return
  }
  for (const payload of payloads) {
    const result = spawnSync("bash", [scriptPath], {
      cwd: directory,
      timeout: VALIDATOR_TIMEOUT_MS,
      input: payload,
      encoding: "utf8",
    })
    if (result.error) {
      console.warn("aispec governance: post-tool validator could not run: " + result.error.message)
      continue
    }
    if (result.status !== 0) {
      const detail = (result.stderr || "").trim()
      console.warn("GOVERNANCE OBSERVED at tool.execute.after: " + (detail || "canonical post-tool validator rejected"))
    }
  }
}

export const GovernancePlugin = async ({ directory }) => {
  return {
    "tool.execute.before": async (input, output) => {
      const tool = input && input.tool
      if (!tool) {
        return
      }
      if (!MUTATING_TOOLS.has(tool)) {
        if (READ_ONLY_TOOLS.has(tool)) {
          return
        }
        if (isOrchestrated()) {
          throw new Error(correctiveMessage(tool, unknownToolDenial(tool)))
        }
        warnUnrecognizedToolOnce(tool)
        return
      }
      const resolved = resolveValidationTargets(tool, output && output.args)
      if (resolved.denial) {
        throw new Error(correctiveMessage(tool, resolved.denial))
      }
      const files = resolved.targets
      const key = files.length > 0 ? memoKey(tool, files) : commandMemoKey(tool, resolved.command)
      const payloads = files.length > 0 ? files.map(hookPayload) : [commandPayload(resolved.command)]
      const cached = memo.get(key)
      if (cached) {
        if (cached.blocked) {
          throw new Error(correctiveMessage(tool, cached.reason))
        }
        return
      }
      const verdict = runCanonicalValidator(directory, payloads)
      memo.set(key, verdict)
      if (verdict.blocked) {
        throw new Error(correctiveMessage(tool, verdict.reason))
      }
    },
    "tool.execute.after": async (input, output) => {
      const tool = (input && input.tool) || ""
      observePostTool(directory, tool, input && input.args)
    },
    event: async ({ event }) => {
      if (event.type !== "session.idle") {
        return
      }
      const verdict = runSessionEndValidator(directory)
      if (verdict.blocked) {
        writeSessionIdleStatus("blocked")
        throw new Error(correctiveMessage("session.idle", verdict.reason))
      }
      writeSessionIdleStatus("allowed")
    },
  }
}

export default GovernancePlugin
