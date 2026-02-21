#!/usr/bin/env node

import fs from "node:fs"

const DENYLIST = [
  /\bif you want\b/i,
  /\bwould you like\b/i,
  /\blet me know if\b/i,
  /\bi can also\b/i,
  /\bif needed\b/i,
  /\bif you prefer\b/i,
  /\byou may want to\b/i,
  /\bi\s+will\b/i,
  /\bi'll\b/i,
  /원하면/,
  /원하시면/,
  /필요하면/,
  /필요하시면/,
  /가능하면/,
  /해줄게/,
  /해드릴게요/,
  /할 수 있어요/,
  /하겠습니다/,
  /겠습니다/,
]

function parseJson(input) {
  try {
    return JSON.parse(input)
  } catch {
    return null
  }
}

function readStdin() {
  return fs.readFileSync(0, "utf8")
}

function extractAssistantTextsFromJsonl(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return []
  const raw = fs.readFileSync(filePath, "utf8")
  const texts = []
  const lines = raw.split(/\r?\n/)

  for (const line of lines) {
    if (!line.trim()) continue
    const obj = parseJson(line)
    if (!obj || typeof obj !== "object") continue

    if (obj.type === "assistant" && obj.message?.role === "assistant") {
      const content = Array.isArray(obj.message.content) ? obj.message.content : []
      for (const part of content) {
        if (part && typeof part === "object" && part.type === "text" && typeof part.text === "string") {
          texts.push(part.text)
        }
      }
    }

    if (obj.role === "assistant" && typeof obj.content === "string") {
      texts.push(obj.content)
    }
  }

  return texts
}

function findDenylistHits(texts) {
  const hits = []
  for (const text of texts) {
    const lines = String(text).split(/\r?\n/)
    for (let i = 0; i < lines.length; i += 1) {
      const line = lines[i]
      for (const pattern of DENYLIST) {
        if (pattern.test(line)) {
          hits.push({ line: i + 1, pattern: pattern.toString(), text: line.trim() })
        }
      }
    }
  }
  return hits
}

function printJson(obj) {
  process.stdout.write(`${JSON.stringify(obj)}\n`)
}

const input = readStdin()
const payload = parseJson(input)

if (!payload || typeof payload !== "object") {
  printJson({ decision: "continue" })
  process.exit(0)
}

const hook = payload.hook_event_name

if (hook === "UserPromptSubmit") {
  const systemMessage = [
    "Execution policy reminder:",
    "- Never use optional/defer phrases (if you want, if needed, 원하면, 필요하면).",
    "- Never use future-intent wording (I will, I'll, 하겠습니다).",
    "- Execute first, then report evidence.",
    "- Do not end with future-intent wording.",
  ].join("\n")
  process.stdout.write(systemMessage)
  process.exit(0)
}

if (hook === "Stop") {
  const texts = extractAssistantTextsFromJsonl(payload.transcript_path)
  const hits = findDenylistHits(texts)

  if (hits.length > 0) {
    const details = hits
      .slice(0, 3)
      .map((hit) => `- ${hit.pattern}: ${hit.text}`)
      .join("\n")

    printJson({
      decision: "block",
      reason: "Autonomy guard blocked defer/optional phrasing in assistant output.",
      inject_prompt: [
        "Rewrite the final response in executed-state phrasing.",
        "Remove optional/defer wording and provide concrete evidence.",
        "Detected lines:",
        details,
      ].join("\n"),
    })
    process.exit(0)
  }

  printJson({ decision: "continue" })
  process.exit(0)
}

printJson({ decision: "continue" })
process.exit(0)
