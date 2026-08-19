---
title: Secrets
id: engineering.security.secrets
---

<!-- alaws:commentary -->

Rules for how agents handle credentials discovered in, or introduced into,
the repository. These rules work alongside
[authentication requirements](alaws:engineering.security.authentication.agent-authenticate-using-short-lived-scoped) -
a leaked credential is useless if auth is short-lived and scoped.

<!-- alaws:laws -->

1. Credentials must never be committed to source control, including in commit messages or code comments. {#credentials-never-committed-source-control}

2. Agents must not print credentials into logs, error messages, or any output that may be persisted. {#agents-print-credentials-into-logs}

3. Credentials discovered in source must be treated as compromised and rotated, not merely removed. {#credentials-discovered-source-treated-compromised}

4. Secrets required at runtime must be retrieved from the approved secret store, never hardcoded. {#secrets-required-runtime-retrieved-approved}
