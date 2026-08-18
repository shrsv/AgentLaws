---
title: Transaction Limits
id: payments.authorization.transaction_limits
---

<!-- alaws:commentary -->

Rules for per-transaction and velocity limits.

<!-- alaws:laws -->

1. A transaction above {{amount}} {{currency}} to merchant {{merchant_id}} must pass step-up verification before it is authorized.

2. An agent must not increase a customer's transaction limit without an explicit, logged customer request.

3. Velocity limits (transactions per hour) must be enforced even when each individual transaction is within its own limit.
