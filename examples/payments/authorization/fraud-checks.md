---
title: Fraud Checks
id: payments.authorization.fraud_checks
---

<!-- alaws:commentary -->

Rules for how an agent handles a transaction flagged by the fraud model.
Fraud checks run in addition to [transaction limit verification](alaws:payments.authorization.transaction_limits.transaction-above-merchant-pass-step-up) - a
transaction must pass both before it is processed.

<!-- alaws:laws -->

1. A transaction flagged by the fraud model must not be auto-approved by an agent, regardless of confidence score. {#transaction-flagged-fraud-model-auto-approved}

2. Agents must not disclose to a customer which specific fraud signal triggered a hold. {#agents-disclose-customer-which-specific}

3. A false positive must be logged with enough detail to retrain the fraud model, not simply overridden and forgotten. {#false-positive-logged-enough-detail}
