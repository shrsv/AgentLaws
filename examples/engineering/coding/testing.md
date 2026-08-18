---
title: Testing
id: engineering.coding.testing
---

<!-- alaws:commentary -->

Rules for what test coverage a change needs before it can be proposed.

<!-- alaws:laws -->

1. A change that modifies behavior must include or update an automated test that would fail without the change.

2. Agents must not disable or skip a failing test to make a build pass without flagging it to a human.

3. Test suites must be run locally or in CI before a change is proposed for review.
