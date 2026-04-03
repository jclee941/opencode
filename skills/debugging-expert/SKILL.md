# Debugging Expert Skill

**Trigger**: Load when encountering errors, bugs, unexpected behavior, or when user requests debugging assistance.

## CORE PHILOSOPHY

Debugging is systematic investigation, not random trial-and-error. Follow the scientific method:
1. **Observe** — Gather symptoms without assumptions
2. **Hypothesize** — Form testable theories about root cause
3. **Test** — Verify/falsify hypotheses with minimal intervention
4. **Conclude** — Fix the actual root cause, not symptoms

## MODE DETECTION

Detect debugging mode from request patterns:

| Pattern | Mode |
|---------|------|
| "error", "exception", "crash", "failing" | ERROR_MODE |
| "slow", "performance", "timeout", "memory" | PERFORMANCE_MODE |
| "wrong output", "unexpected", "should be" | LOGIC_MODE |
| "intermittent", "sometimes", "flaky", "race" | TIMING_MODE |
| "works locally", "only in prod", "environment" | ENVIRONMENT_MODE |

## PHASE 0: CONTEXT GATHERING (PARALLEL)

Fire these in parallel before any hypothesis:

```
1. Read error message/stack trace (EXACT text, not paraphrase)
2. Identify error location (file:line)
3. Check recent changes (git diff HEAD~5, git log -10 --oneline)
4. Search codebase for similar patterns (grep, ast-grep)
5. Check logs if available (application logs, system logs)
```

**CRITICAL**: Capture the EXACT error message. Paraphrased errors lose diagnostic value.

## PHASE 1: SYMPTOM ANALYSIS

### Error Classification

| Error Type | Investigation Path |
|------------|-------------------|
| SyntaxError/ParseError | Check recent edits, linter output |
| TypeError/NullPointer | Trace data flow to null source |
| ImportError/ModuleNotFound | Check paths, dependencies, versions |
| NetworkError/Timeout | Check connectivity, DNS, firewall |
| PermissionError | Check file/resource permissions |
| OutOfMemory | Profile memory usage, check leaks |
| AssertionError/TestFail | Compare expected vs actual values |

### Stack Trace Reading

1. **Bottom-up**: Start from the deepest frame (actual error location)
2. **Your code first**: Find the first frame in YOUR codebase (not library code)
3. **Boundary crossing**: Note where control passes between your code and libraries
4. **Async boundaries**: Identify promise chains, callback boundaries, goroutine spawns

## PHASE 2: HYPOTHESIS FORMATION

### Hypothesis Quality Checklist

A good hypothesis is:
- [ ] **Specific**: Points to a concrete code location or condition
- [ ] **Testable**: Can be verified/falsified with a single experiment
- [ ] **Explains symptoms**: Accounts for ALL observed behavior
- [ ] **Minimal**: Doesn't assume more than necessary (Occam's Razor)

### Common Root Cause Patterns

| Symptom | Common Causes |
|---------|---------------|
| "It was working yesterday" | Recent commit, dependency update, config change |
| "Works on my machine" | Environment diff, missing env var, path issue |
| "Only fails sometimes" | Race condition, timing dependency, external state |
| "Fails only with real data" | Edge case, encoding issue, data validation gap |
| "Fails after N iterations" | Resource leak, accumulating state, counter overflow |

## PHASE 3: ISOLATION & TESTING

### Isolation Strategies

1. **Binary search**: Bisect code/commits to narrow down cause
2. **Minimal reproduction**: Create smallest failing case
3. **Dependency elimination**: Remove components until failure stops
4. **Environment reset**: Fresh environment to rule out state

### Testing Hypotheses

For each hypothesis, define:
```
IF [hypothesis] THEN [expected observation]
TEST: [concrete action to verify]
RESULT: [actual observation]
VERDICT: CONFIRMED / REFUTED / INCONCLUSIVE
```

### Git Bisect for Regression

```bash
git bisect start
git bisect bad HEAD
git bisect good <last-known-good-commit>
# Run test at each step
git bisect run <test-command>
```

## PHASE 4: FIX IMPLEMENTATION

### Fix Quality Rules

1. **Root cause only**: Fix the actual cause, not symptoms
2. **Minimal change**: Smallest diff that resolves the issue
3. **No collateral refactoring**: Don't "clean up" while fixing bugs
4. **Add regression test**: Prevent recurrence with specific test case
5. **Document the fix**: Explain WHY in commit message, not just WHAT

### Fix Verification

```
1. Confirm original error no longer occurs
2. Run existing test suite (catch regressions)
3. Test edge cases related to the fix
4. Check related code paths
```

## MODE-SPECIFIC PROTOCOLS

### ERROR_MODE Protocol

```
1. Capture exact error message and full stack trace
2. Identify error type and classification
3. Trace to first frame in your code
4. Read code at that location
5. Trace data flow to find where bad state originated
6. Form hypothesis about root cause
7. Test hypothesis
8. Fix root cause
9. Add regression test
```

### PERFORMANCE_MODE Protocol

```
1. Quantify the problem (N ms/seconds, X% CPU, Y MB memory)
2. Identify WHERE (profiler, flame graph, metrics)
3. Identify WHEN (startup, under load, specific operation)
4. Check for common issues:
   - N+1 queries
   - Missing indexes
   - Unbounded loops
   - Memory leaks (object accumulation)
   - Blocking I/O in async context
   - Excessive allocations
5. Profile before and after fix
6. Verify improvement with metrics
```

### LOGIC_MODE Protocol

```
1. Define expected behavior precisely
2. Define actual behavior precisely
3. Identify divergence point
4. Add logging/breakpoints at divergence point
5. Trace variable values through execution
6. Find where actual diverges from expected
7. Fix logic at divergence point
8. Add test case covering this scenario
```

### TIMING_MODE Protocol

```
1. Identify timing-dependent components
2. Check for:
   - Shared mutable state
   - Missing synchronization
   - Order-dependent initialization
   - Timeout races
   - Cache invalidation timing
3. Add logging with timestamps
4. Reproduce under stress (high concurrency)
5. Apply appropriate synchronization
6. Verify with stress testing
```

### ENVIRONMENT_MODE Protocol

```
1. Document exact environment where it works
2. Document exact environment where it fails
3. Diff environments:
   - OS version
   - Language runtime version
   - Dependency versions (lock file diff)
   - Environment variables
   - Configuration files
   - Network topology
4. Identify the difference causing failure
5. Either fix the code or document requirement
```

## TOOL USAGE

### Primary Tools

| Tool | Use Case |
|------|----------|
| `lsp_diagnostics` | Type errors, lint issues |
| `grep/ast-grep` | Pattern search in codebase |
| `bash` | Run tests, check logs, git operations |
| `read` | Examine code, config files |
| `pty_spawn` | Interactive debugging sessions |

### Debugging Commands

```bash
# Search for error message in codebase
grep -r "error message text" .

# Find recent changes to problematic file
git log -p --follow -- path/to/file

# Check dependency versions
npm list | grep <package>  # Node
pip show <package>          # Python
go list -m all | grep <module>  # Go

# Find where symbol is defined
grep -rn "function symbolName" .
ast-grep -p 'function $NAME($$$)' -l javascript
```

## ANTI-PATTERNS

### Investigation Anti-Patterns

- **Shotgun debugging**: Random changes hoping something works
- **Assuming the cause**: Acting without verifying hypothesis
- **Ignoring error messages**: They usually tell you exactly what's wrong
- **Only reading happy path**: Missing error handlers, edge cases
- **Surface fixes**: Catching/suppressing errors without fixing cause

### Fix Anti-Patterns

- **Cargo cult fixes**: Copying solution without understanding why it works
- **Defensive overcompensation**: Adding excessive null checks everywhere
- **TODO-driven fixes**: "TODO: fix this properly later" (never happens)
- **Test deletion**: Removing failing tests instead of fixing code
- **Magic numbers**: Hardcoding values to make tests pass

## OUTPUT FORMAT

When debugging, structure output as:

```
## Symptom
[Exact error message and context]

## Investigation
[Steps taken, tools used, observations]

## Root Cause
[Specific cause with evidence]

## Fix
[Minimal change applied]

## Verification
[Test results confirming fix]

## Prevention
[Regression test added or recommendation]
```

## ESCALATION

Escalate to Oracle when:
- 3+ failed hypotheses
- Unfamiliar domain/framework
- Architectural root cause requiring redesign
- Security-sensitive bug needing expert review
