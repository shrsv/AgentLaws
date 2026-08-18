---
title: Dependencies
id: engineering.security.dependencies
---

<!-- alaws:commentary -->

Rules for adding, upgrading, and evaluating third-party dependencies.

<!-- alaws:laws -->

1. Before adding a new dependency to {{repo}}, an agent must check it for known vulnerabilities using the approved scanner.

2. Agents must not upgrade a dependency across a major version without flagging the change for human review.

3. Dependencies with no maintenance activity in the last two years must be flagged as a risk, not silently relied upon.
