#!/usr/bin/env node

import { readdir } from "node:fs/promises";
import path from "node:path";

const rootDir = process.cwd();

const ignoreDirs = new Set([
  ".git",
  "node_modules",
  "data",
  "log",
  "logs",
  "tmp",
  "profiles",
  ".sisyphus",
  ".cache",
  "dist",
  "coverage",
  ".next",
  ".venv",
  ".ruff_cache",
]);

const allowedUppercaseFiles = new Set([
  "AGENTS.md",
  "SKILL.md",
  "README.md",
  "CHANGELOG.md",
  "LICENSE",
  "BUILD",
  "BUILD.bazel",
  "WORKSPACE",
  "WORKSPACE.bazel",
  "MODULE.bazel",
  "OWNERS",
]);

const allowedSpecialDirs = new Set([
  "__tests__",
  "__snapshots__",
  "__fixtures__",
]);

const dirNamePattern = /^[a-z0-9][a-z0-9_-]*$/;
const fileNamePattern = /^[a-z0-9][a-z0-9._-]*$/;
const backupPattern = /^.+\.backup-\d{8}-\d{6}(?:-\d{3}Z)?$/;

const violations = [];

function toRel(fullPath) {
  return path.relative(rootDir, fullPath) || ".";
}

async function walk(currentDir) {
  const entries = await readdir(currentDir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = path.join(currentDir, entry.name);
    const relPath = toRel(fullPath);

    if (entry.isDirectory()) {
      if (ignoreDirs.has(entry.name)) {
        continue;
      }

      if (allowedSpecialDirs.has(entry.name)) {
        await walk(fullPath);
        continue;
      }

      if (!entry.name.startsWith(".") && !dirNamePattern.test(entry.name)) {
        violations.push(`DIR  ${relPath}`);
      }

      await walk(fullPath);
      continue;
    }

    if (entry.isFile()) {
      if (entry.name.startsWith(".")) {
        continue;
      }

      if (allowedUppercaseFiles.has(entry.name)) {
        continue;
      }

      if (backupPattern.test(entry.name)) {
        continue;
      }

      if (!fileNamePattern.test(entry.name)) {
        violations.push(`FILE ${relPath}`);
      }
    }
  }
}

await walk(rootDir);

if (violations.length > 0) {
  console.error("Naming rule violations found:\n");
  for (const line of violations) {
    console.error(`- ${line}`);
  }
  console.error(`\nTotal violations: ${violations.length}`);
  process.exit(1);
}

console.log("Naming rules check passed.");
