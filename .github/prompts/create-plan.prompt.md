---
name: create-plan
description: "Create a detailed implementation plan through interactive research and iteration"
agent: planner
argument-hint: "Describe what you want to build, or provide a path to a research document"
---

# Create Implementation Plan

Create a detailed implementation plan through an interactive, iterative process.

## Your Task

Plan: **$input**

## Process

### Step 1: Context Gathering

1. **Read any provided files immediately and FULLY**:
   - Research documents
   - Related implementation plans
   - Any context files mentioned

2. **Gather codebase context using sub-agents** (if available):

   Run the **codebase-locator** agent as a subagent:

   > "Find all files related to [task]. Return a categorized listing."

   Run the **codebase-analyzer** agent as a subagent:

   > "Analyze how the current implementation works in [area]. Return file:line references."

   Run the **pattern-finder** agent as a subagent:

   > "Find similar implementations we can model after. Return concrete examples."

   **If sub-agents are not available**, perform research directly using search tools.

3. **Present your understanding** with specific questions:

   ```
   Based on the context and my research, I understand we need to [summary].

   I've found that:
   - [Current implementation detail with file:line reference]
   - [Relevant pattern or constraint discovered]

   Questions my research couldn't answer:
   - [Specific technical question requiring human judgment]
   ```

### Step 2: Research & Discovery

1. **Verify any corrections** - don't just accept them, research to confirm
2. **Present design options** with pros/cons
3. **Get alignment** on approach before writing the plan

### Step 3: Plan Structure

1. **Propose outline** and get feedback:

   ```
   ## Implementation Phases:
   1. [Phase name] - [what it accomplishes]
   2. [Phase name] - [what it accomplishes]
   ```

2. **Get approval** on structure before writing details

### Step 4: Write Detailed Plan

Write to `thoughts/shared/plans/YYYY-MM-DD-description.md`:

```markdown
# [Feature/Task Name] Implementation Plan

## Overview

[Brief description]

## Current State Analysis

[What exists now, constraints discovered]

## Desired End State

[Specification and verification approach]

### Key Discoveries:

- [Finding with file:line reference]
- [Pattern to follow]

## What We're NOT Doing

[Out-of-scope items]

## Implementation Approach

[High-level strategy]

## Phase 1: [Descriptive Name]

### Overview

[What this phase accomplishes]

### Changes Required:

#### 1. [Component/File Group]

**File**: `path/to/file`
**Changes**: [Summary]
```

# Specific code to add/modify

```

### Success Criteria:

#### Automated Verification:
- [ ] Tests pass using the project's standard automated test command (see language-specific instructions)
- [ ] Type or interface checks pass using the prescribed verifier
- [ ] Linting or static analysis passes using the required toolchain
- [ ] Formatting matches project conventions using the approved formatter

#### Manual Verification:
- [ ] Feature works as expected
- [ ] Edge cases verified

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 2: [Descriptive Name]
[Similar structure...]

---

## Testing Strategy

### Unit Tests:
- [What to test]

### Integration Tests:
- [End-to-end scenarios]

### Manual Testing Steps:
1. [Verification step]

## References
- Research: `thoughts/shared/research/[relevant].md`
- Similar implementation: `[file:line]`
```

### Step 5: Review and Iterate

Present the draft and iterate based on feedback.

## Critical Guidelines

1. **Be Skeptical**: Question vague requirements, verify with code
2. **Be Interactive**: Get buy-in at each step, don't write everything at once
3. **Be Thorough**: Include specific file paths and line numbers
4. **Be Practical**: Incremental, testable changes with clear scope
5. **No Open Questions**: Research or ask for clarification immediately

## Success Criteria Guidelines

**Always separate into two categories:**

1. **Automated Verification**: Run the language-specific automated tests, linting/static analysis, type or interface checks, and formatting commands defined in the repository instructions
2. **Manual Verification**: UI testing, edge cases, performance
