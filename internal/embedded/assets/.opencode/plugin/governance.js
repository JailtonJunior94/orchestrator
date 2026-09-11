import { existsSync, writeFileSync } from "node:fs"
import { spawnSync } from "node:child_process"
import { join } from "node:path"

const SENTINEL_ENV_VAR = "AISPEC_OPENCODE_GOVERNANCE_SENTINEL"
const ORCHESTRATED_ENV_VAR = "AISPEC_OPENCODE_ORCHESTRATED"
const CANONICAL_PRE_TOOL_SCRIPT = ".agents/scripts/hook-prereq-gate.sh"
const CANONICAL_SESSION_END_SCRIPT = ".agents/scripts/validate-session-end.sh"
const MUTATING_TOOLS = new Set(["bash", "write", "edit", "multiedit", "patch"])
const VALIDATOR_TIMEOUT_MS = Number(process.env.AISPEC_OPENCODE_VALIDATOR_TIMEOUT_MS) || 8000

const sentinelPath = process.env[SENTINEL_ENV_VAR]
if (sentinelPath) {
  writeFileSync(sentinelPath, "")
}

const memo = new Map()
let validatorExistsCache = null
let sessionEndValidatorExistsCache = null
let interactiveWarnedOnce = false
let sessionEndInteractiveWarnedOnce = false
const unrecognizedToolWarned = new Set()

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
  if (!args || typeof args !== "object") {
    return []
  }
  const files = []
  for (const key of ["filePath", "path", "file"]) {
    const value = args[key]
    if (typeof value === "string" && value.length > 0) {
      files.push(value)
    }
  }
  return files
}

function memoKey(tool, files) {
  return tool + "::" + files.slice().sort().join(",")
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

function sessionEndAdvisoryMessage(reason) {
  return "GOVERNANCE ADVISORY at session.idle: " + reason +
    "\nThis point is observational only — only tool.execute.before blocks (RF-19/RF-27) — recorded for visibility, session was not aborted."
}

function runCanonicalValidator(directory, files) {
  const scriptPath = join(directory, CANONICAL_PRE_TOOL_SCRIPT)
  if (!validatorExists(scriptPath)) {
    const orchestrated = process.env[ORCHESTRATED_ENV_VAR] === "1"
    if (orchestrated) {
      return { blocked: true, reason: "canonical validator " + CANONICAL_PRE_TOOL_SCRIPT + " is missing — run 'ai-spec-harness install .' before retrying" }
    }
    if (!interactiveWarnedOnce) {
      interactiveWarnedOnce = true
      console.warn("aispec governance: canonical validator missing at " + scriptPath + " — proceeding without gate (interactive mode, warned once)")
    }
    return { blocked: false }
  }
  const result = spawnSync("bash", [scriptPath, ...files], {
    cwd: directory,
    timeout: VALIDATOR_TIMEOUT_MS,
    input: "",
    encoding: "utf8",
  })
  if (result.error && result.error.code === "ETIMEDOUT") {
    return { blocked: true, reason: "validator timed out after " + VALIDATOR_TIMEOUT_MS + "ms — timeout is treated as denial, never approval" }
  }
  if (result.status !== 0) {
    const detail = (result.stderr || "").trim()
    return { blocked: true, reason: detail.length > 0 ? detail : "canonical validator rejected this tool call" }
  }
  return { blocked: false }
}

function runSessionEndValidator(directory) {
  const scriptPath = join(directory, CANONICAL_SESSION_END_SCRIPT)
  if (!sessionEndValidatorExists(scriptPath)) {
    const orchestrated = process.env[ORCHESTRATED_ENV_VAR] === "1"
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

export const GovernancePlugin = async ({ directory }) => {
  return {
    "tool.execute.before": async (input, output) => {
      const tool = input && input.tool
      if (!tool) {
        return
      }
      if (!MUTATING_TOOLS.has(tool)) {
        warnUnrecognizedToolOnce(tool)
        return
      }
      const files = extractTouchedFiles(output && output.args)
      const key = memoKey(tool, files)
      const cached = memo.get(key)
      if (cached) {
        if (cached.blocked) {
          throw new Error(correctiveMessage(tool, cached.reason))
        }
        return
      }
      const verdict = runCanonicalValidator(directory, files)
      memo.set(key, verdict)
      if (verdict.blocked) {
        throw new Error(correctiveMessage(tool, verdict.reason))
      }
    },
    "tool.execute.after": async () => {},
    "session.idle": async () => {
      const verdict = runSessionEndValidator(directory)
      if (verdict.blocked) {
        console.warn(sessionEndAdvisoryMessage(verdict.reason))
      }
    },
  }
}

export default GovernancePlugin
