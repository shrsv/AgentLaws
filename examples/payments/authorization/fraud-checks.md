---
title: Fraud Checks
id: payments.authorization.fraud_checks
---

<!-- alaws:commentary -->

Rules for how an agent handles a transaction flagged by the fraud model.

<!-- alaws:laws -->

1. A transaction flagged by the fraud model must not be auto-approved by an agent, regardless of confidence score.

2. Agents must not disclose to a customer which specific fraud signal triggered a hold.

3. A false positive must be logged with enough detail to retrain the fraud model, not simply overridden and forgotten.
