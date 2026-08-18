---
title: Monitoring
id: engineering.operations.monitoring
---

<!-- alaws:commentary -->

Rules for alerting and anomaly handling once a service is live.

<!-- alaws:laws -->

1. A new production service must have baseline alerting configured before it receives real traffic.

2. Agents must not silence an alert without recording the reason and an expiry for the silence.

3. Anomalies detected by an agent must be reported even if the agent itself is not the cause.
