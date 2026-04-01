---
name: implement-plan
description: "Implement a technical plan phase by phase with verification"
agent: implementer
argument-hint: 'Path to plan file (e.g., "thoughts/shared/plans/2025-02-02-feature.md")'
---

# Implement Plan

Implement an approved technical plan from `thoughts/shared/plans/` phase by phase with verification.

## Your Task

Implement: **$input**

## Getting Started

1. **Read the plan completely** and check for any existing checkmarks (`- [x]`)
2. **Read all files mentioned** in the plan
3. **Read files fully** - never partially, you need complete context
4. **Think deeply** about how the pieces fit together
5. **Start implementing** if you understand what needs to be done

If no plan path provided, ask for one.

## Implementation Workflow

### For Each Phase:

1. **Read all relevant files** mentioned in the phase

2. **Make the required changes** as specified in the plan

3. **Run automated verification**:
   - Execute the project's standard automated tests (see the language-specific instructions)
   - Run the required linting or static analysis tools
   - Apply or verify formatting with the approved formatter
   - Perform the mandated type or interface checks

4. **Fix any issues** before proceeding

5. **Update the plan**:
   - Check off completed items using the edit tool
   - Mark automated verification items as done

6. **Pause for human verification**:

   ```
   Phase [N] Complete - Ready for Manual Verification

   Automated verification passed:
   - [List automated checks that passed]

   Please perform the manual verification steps listed in the plan:
   - [List manual verification items from the plan]

   Let me know when manual testing is complete so I can proceed to Phase [N+1].
   ```

7. **Only proceed** to the next phase after confirmation

**Important**: Do NOT check off manual testing items until confirmed by the user.

## When Things Don't Match

Plans are carefully designed, but reality can be messy. If you encounter a mismatch:

**STOP and report instead of improvising:**

```
Issue in Phase [N]:
Expected: [what the plan says]
Found: [actual situation]
Why this matters: [explanation]

How should I proceed?
```

## If You Get Stuck

1. **Make sure you've read and understood all relevant code**
2. **Consider if the codebase has evolved** since the plan was written
3. **Present the mismatch clearly** and ask for guidance

**If sub-agents are available**, delegate debugging:

Run the **codebase-analyzer** agent:

> "Analyze how [component] works and why [issue] might be happening. Return file:line references."

Run the **pattern-finder** agent:

> "Find how similar issues were solved elsewhere in the codebase."

**If sub-agents are not available**, investigate directly using search tools.

## Resuming Work

If the plan has existing checkmarks:

- Trust that completed work is done
- Pick up from the first unchecked item
- Verify previous work only if something seems off

## Critical Guidelines

1. **Follow the plan's intent** - understand the goal, not just the steps
2. **Report, don't improvise** - if something doesn't match, ask before changing approach
3. **One phase at a time** - complete and verify before moving on
4. **Human gates matter** - the pause between phases is intentional
5. **Update the plan** - check off items as you complete them
6. **Test thoroughly** - run all verification commands

## Verification Commands Reference

Use the language-specific repository instructions to determine the exact commands for:

- Running the full automated test suite and any focused subsets
- Executing linting or static analysis
- Applying or verifying formatting
- Performing type or interface checks

Remember: You're implementing a solution, not just checking boxes. Keep the end goal in mind and maintain forward momentum while respecting the human-gated workflow.
