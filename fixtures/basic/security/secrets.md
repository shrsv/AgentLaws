---
title: Security
id: engineering.security
---

<!-- alaws:commentary -->

This section defines the security requirements for agents working with the
repository.

The commentary explains rationale, trade-offs, history, examples, and
anything useful to the people maintaining the lawbook.

<!-- alaws:laws -->

1. Credentials must never be committed to source control.

2. Agents must not print credentials into logs.

3. Credentials discovered in source must be treated as compromised.
