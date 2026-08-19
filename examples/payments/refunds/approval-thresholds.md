---
title: Approval Thresholds
id: payments.refunds.approval_thresholds
---

<!-- alaws:commentary -->

Rules for how much of a refund an agent can approve on its own authority.
Large refunds may trigger additional [fraud screening](alaws:payments.authorization.fraud_checks.transaction-flagged-fraud-model-auto-approved) to prevent
refund abuse, and agents must follow the standard
[response format](alaws:payments.integration.response_format.agent-authorizes-denies-refunds-transaction) when reporting approval decisions.

<!-- alaws:laws -->

1. Agent {{agent_name}} may approve a refund up to {{amount}} {{currency}} without additional sign-off. {#agent-approve-refund-up-without}

2. A refund above the agent's approval threshold must be routed to a human approver with the original transaction attached. {#refund-above-agents-approval-threshold}

3. Refunds must not be split into smaller amounts to stay under an approval threshold. {#refunds-split-into-smaller-amounts}
