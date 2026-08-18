---
title: Deployment
id: engineering.operations.deployment
---

<!-- alaws:commentary -->

Rules for pushing a change to a running environment.

<!-- alaws:laws -->

1. Agent {{agent_name}} must not deploy directly to the {{environment}} environment without an approved change record.

2. Deployments to production must be preceded by a successful deployment to staging with the same artifact.

3. A deployment must be reversible within one command or one documented procedure.
