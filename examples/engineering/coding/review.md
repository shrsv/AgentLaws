---
title: Code Review
id: engineering.coding.review
---

<!-- alaws:commentary -->

Rules for how a code change gets reviewed before it merges. A reviewer
should verify that [testing obligations](alaws:engineering.coding.testing.change-modifies-behavior-include-update) have been met and that
[secrets are not introduced](alaws:engineering.security.secrets.credentials-never-committed-source-control) into the change.

<!-- alaws:laws -->

1. Every change to {{repo}} must be reviewed by at least one human, {{reviewer}} or another qualified reviewer, before merging. {#every-change-reviewed-least-one}

2. An agent must not approve its own pull request. {#agent-approve-own-pull-request}

3. Review comments that request a change must be resolved or explicitly declined with a rationale before merge. {#review-comments-request-change-resolved}
