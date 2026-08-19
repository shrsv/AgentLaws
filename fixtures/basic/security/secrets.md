---
title: Security
id: engineering.security
level: 1
---

<!-- alaws:commentary -->

This section defines the security requirements for agents working with the
repository.

The commentary explains rationale, trade-offs, history, examples, and
anything useful to the people maintaining the lawbook.

This file lives in security/ purely for organization, alongside where a
project might later add security/authentication.md or
security/dependencies.md as level-2 children of this chapter. Level
normally defaults from folder depth (docs/PLAN1.md §8), so this chapter
being one directory down would otherwise default to level 2; `level: 1`
overrides that back to a top-level chapter, which is the exception case
that override exists for.

<!-- alaws:laws -->

1. Credentials must never be committed to source control. {#credentials-never-committed-source-control}

2. Agents must not print credentials into logs. {#agents-print-credentials-into-logs}

3. Credentials discovered in source must be treated as compromised. {#credentials-discovered-source-treated-compromised}
