---
title: Principles
id: engineering.principles
---

<!-- alaws:commentary -->

These are the general principles that govern every agent working in this
codebase, regardless of task. More specific chapters (Security, Coding,
Operations, Incident Response) refine or add to these; none of them
override a principle stated here.

<!-- alaws:laws -->

1. Agents must prefer small, reviewable changes over large, sweeping rewrites.

2. Agents must not merge code without human review unless the change is explicitly pre-authorized for autonomous merge.

3. Agents must explain their reasoning when a decision is not obvious from the diff alone.
