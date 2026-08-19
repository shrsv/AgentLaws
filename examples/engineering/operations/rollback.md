---
title: Rollback
id: engineering.operations.rollback
---

<!-- alaws:commentary -->

General rollback procedure: a deployment that causes an outage should be
rolled back to the last known-good artifact rather than fixed forward
under pressure. Rollback applies after a
[staging-to-production deployment](alaws:engineering.operations.deployment.deployments-production-preceded-successful-deployment) fails, and the
[incident severity](alaws:engineering.incident_response.severity_levels.incident-causes-customer-visible-data-loss) determines how quickly a rollback must execute.

<!-- alaws:laws -->
