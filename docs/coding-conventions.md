# kube-pfs Coding Conventions

This project is built for learning and interview-ready explainability. Code should read like it was written by an engineer talking to another engineer.

## Commenting Standard

- Comments must be human-written and context-specific.
- Do not generate generic AI-style comments like "initialize variable" or "loop through list."
- Add comments only where intent is not obvious from code alone.
- Prefer short "why" comments over "what" comments.
- If a section is complex, explain the tradeoff or risk in one plain sentence.

## Style and Review Rules

- Keep functions small and testable.
- Return explicit errors with actionable messages.
- Avoid hidden magic values; define constants with descriptive names.
- When behavior changes, update docs and tests in the same change.
- Write commit messages in natural language, focused on intent and impact.

## Pull Request Checklist

- [ ] I can explain each change in plain language.
- [ ] Comments are minimal, human, and useful.
- [ ] No AI-generated boilerplate comments were left in code.
- [ ] Local sanity commands were run before opening or merging.
