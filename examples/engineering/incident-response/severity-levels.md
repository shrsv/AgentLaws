---
title: Severity Levels
id: engineering.incident_response.severity_levels
---

<!-- alaws:commentary -->

How an incident's severity is assigned and revised. Severity directly
governs [communication cadence](alaws:engineering.incident_response.communication.incident-severity-status-update-posted) and determines whether
automated [rollback](alaws:engineering.operations.rollback) should be triggered.

<!-- alaws:laws -->

1. An incident that causes customer-visible data loss must be classified Severity 1 regardless of the number of customers affected. {#incident-causes-customer-visible-data-loss}

2. Severity classification must be reassessed if new information changes the blast radius, not fixed at first report. {#severity-classification-reassessed-new-information}

3. Agents must not downgrade a severity level without a human confirming the downgrade. {#agents-downgrade-severity-level-without}
