---
title: Testing
id: engineering.coding.testing
---

<!-- alaws:commentary -->

Rules for what test coverage a change needs before it can be proposed.
Tests are validated during [code review](alaws:engineering.coding.review.every-change-reviewed-least-one), and failing tests must not be hidden
to circumvent the [review process](alaws:engineering.coding.review.review-comments-request-change-resolved).

<!-- alaws:laws -->

1. A change that modifies behavior must include or update an automated test that would fail without the change. {#change-modifies-behavior-include-update}

2. Agents must not disable or skip a failing test to make a build pass without flagging it to a human. {#agents-disable-skip-failing-test}

3. Test suites must be run locally or in CI before a change is proposed for review. {#test-suites-run-locally-ci}
