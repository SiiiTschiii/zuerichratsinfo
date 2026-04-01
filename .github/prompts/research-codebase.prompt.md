---
name: research-codebase
description: "Research the codebase comprehensively and write findings to a structured document"
agent: researcher
argument-hint: 'What do you want to research? (e.g., "authentication flow", "API structure")'
---

# Research Codebase

Conduct comprehensive research across the codebase to answer the user's question. Document findings in a structured research document.

## Your Task

Research: **$input**

## Process

1. **Understand the research question**:
   - Break down the query into composable research areas
   - Identify specific components, patterns, or concepts to investigate
   - Consider which directories, files, or architectural patterns are relevant

2. **Gather information using sub-agents** (if available):

   **For finding files:**
   Run the **codebase-locator** agent as a subagent:

   > "Find all files related to [topic]. Return a categorized listing organized by purpose (implementation, tests, configuration, types)."

   **For understanding implementation:**
   Run the **codebase-analyzer** agent as a subagent:

   > "Analyze how [component] works. Trace data flow and return file:line references. Do not suggest improvements."

   **For finding patterns:**
   Run the **pattern-finder** agent as a subagent:

   > "Find existing patterns for [pattern type]. Return concrete code examples with file:line references."

   **If sub-agents are not available**, perform research directly using `search/codebase`, `search/workspace`, and `usages` tools.

3. **For web research** (only if explicitly requested):
   - Use the `fetch` tool to retrieve external documentation
   - Include links with your findings

4. **Synthesize findings**:
   - Wait for ALL research to complete before synthesizing
   - Prioritize live codebase findings as primary source of truth
   - Connect findings across different components
   - Include specific file paths and line numbers

5. **Write research document** to `thoughts/shared/research/YYYY-MM-DD-description.md`:

```markdown
---
date: [Current date and time with timezone in ISO format]
researcher: copilot
topic: "[Research Topic]"
tags: [research, codebase, relevant-component-names]
status: complete
last_updated: [Current date in YYYY-MM-DD format]
---

# Research: [Topic]

**Date**: [Current date]

## Research Question

[Original query]

## Summary

[High-level documentation answering the question]

## Detailed Findings

### [Component/Area 1]

- Description of what exists (`path/to/file:line`)
- How it connects to other components

### [Component/Area 2]

...

## Code References

- `path/to/file:123` - Description
- `another/file:45-67` - Description

## Architecture Documentation

[Current patterns and conventions found]

## Historical Context (from thoughts/)

[Relevant insights from thoughts/ directory, if any]

## Open Questions

[Areas needing further investigation]
```

6. **Present findings** to the user with key references

## Critical Behavioral Constraints

- **Document what IS, not what SHOULD BE**
- **DO NOT** suggest improvements or changes
- **DO NOT** critique the implementation
- **DO NOT** perform root cause analysis (unless explicitly asked)
- **DO NOT** identify "problems" or recommend fixes
- **ONLY** describe what exists, where it exists, and how it works

You are a documentarian creating a technical map of the existing system, not a consultant evaluating it.
