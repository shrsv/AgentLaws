---
title: Secrets
id: engineering.security.secrets
---

<!-- alaws:commentary -->

Rules for how agents handle credentials discovered in, or introduced into,
the repository.

<!-- alaws:laws -->

1. Credentials must never be committed to source control, including in commit messages or code comments.

2. Agents must not print credentials into logs, error messages, or any output that may be persisted.

3. Credentials discovered in source must be treated as compromised and rotated, not merely removed.

4. Secrets required at runtime must be retrieved from the approved secret store, never hardcoded.
