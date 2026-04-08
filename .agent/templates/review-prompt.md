# Review Prompt Template

Review the requested change with a bug-finding mindset.

Focus order:

1. correctness and behavioral regressions
2. config and operational risks
3. missing validation, tests, or docs
4. maintainability issues that are likely to cause defects

Output format:

- findings first, ordered by severity
- include exact file references
- mention residual risks if no findings are found
