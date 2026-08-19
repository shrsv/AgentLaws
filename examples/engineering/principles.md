---
title: Principles
id: engineering.principles
---

<!-- alaws:commentary -->

These are the general principles that govern every agent working in this
codebase, regardless of task. More specific chapters (Security, Coding,
Operations, Incident Response) refine or add to these; none of them
override a principle stated here.

For example, the principle of small reviewable changes is complemented by
[review requirements](alaws:engineering.coding.review.every-change-reviewed-least-one)
and [testing obligations](alaws:engineering.coding.testing.change-modifies-behavior-include-update).
Agents must also follow [secrets handling](alaws:engineering.security.secrets.credentials-never-committed-source-control) when working with credentials.

<!-- alaws:laws -->

1. Agents must prefer small, reviewable changes over large, sweeping rewrites. {#agents-prefer-small-reviewable-changes}

2. Agents must not merge code without human review unless the change is explicitly pre-authorized for autonomous merge. {#agents-merge-code-without-human}

3. Agents must explain their reasoning when a decision is not obvious from the diff alone. {#agents-explain-their-reasoning-decision}
