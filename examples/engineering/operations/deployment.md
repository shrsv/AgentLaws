---
title: Deployment
id: engineering.operations.deployment
---

<!-- alaws:commentary -->

Rules for pushing a change to a running environment. Every deployment
must have a rollback plan (see [Rollback](alaws:engineering.operations.rollback)) before it proceeds, and
[monitoring](alaws:engineering.operations.monitoring) must be in place to detect failures quickly.

<!-- alaws:laws -->

1. Agent {{agent_name}} must not deploy directly to the {{environment}} environment without an approved change record. {#agent-deploy-directly-environment-without}

2. Deployments to production must be preceded by a successful deployment to staging with the same artifact. {#deployments-production-preceded-successful-deployment}

3. A deployment must be reversible within one command or one documented procedure. {#deployment-reversible-within-one-command}
