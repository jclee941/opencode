#!/usr/bin/env node

import fs from 'node:fs'

const denylist = [
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

function usage() {
  console.error('Usage: node scripts/lint-assistant-phrasing.mjs <session-export.json>')
  process.exit(2)
}

const file = process.argv[2]
if (!file) usage()

let raw
try {
  raw = fs.readFileSync(file, 'utf8')
} catch (error) {
  console.error(`Failed to read file: ${file}`)
  console.error(String(error))
  process.exit(2)
}

let json
try {
  json = JSON.parse(raw)
} catch (error) {
  console.error(`Invalid JSON: ${file}`)
  console.error(String(error))
  process.exit(2)
}

function* collectAssistantTexts(node) {
  if (!node || typeof node !== 'object') return

  if (Array.isArray(node)) {
    for (const item of node) yield* collectAssistantTexts(item)
    return
  }

  const role = typeof node.role === 'string' ? node.role.toLowerCase() : ''
  if (role === 'assistant') {
    if (typeof node.content === 'string') {
      yield node.content
    } else if (Array.isArray(node.content)) {
      for (const item of node.content) {
        if (typeof item === 'string') {
          yield item
        } else if (item && typeof item.text === 'string') {
          yield item.text
        }
      }
    }
  }

  for (const value of Object.values(node)) {
    if (value && typeof value === 'object') yield* collectAssistantTexts(value)
  }
}

const hits = []
let messageIndex = 0
for (const text of collectAssistantTexts(json)) {
  messageIndex += 1
  const lines = text.split(/\r?\n/)
  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i]
    for (const pattern of denylist) {
      if (pattern.test(line)) {
        hits.push({
          messageIndex,
          line: i + 1,
          pattern: pattern.toString(),
          text: line.trim(),
        })
      }
    }
  }
}

if (hits.length === 0) {
  console.log('assistant-phrasing-lint: ok')
  process.exit(0)
}

console.error('assistant-phrasing-lint: failed')
for (const hit of hits) {
  console.error(
    `message#${hit.messageIndex} line#${hit.line} ${hit.pattern} -> ${hit.text}`,
  )
}
process.exit(1)
