---
title: Emergency Procedures
id: engineering.operations.rollback.emergency
---

<!-- alaws:commentary -->

This section exists three levels deep in the lawbook (Operations >
Rollback > Emergency Procedures), and its citations reflect that:
1.x.y.z-style numbers. It covers the narrow exception to the normal
rollback and deployment rules that applies only during a declared
incident.

<!-- alaws:laws -->

1. During incident {{incident_id}}, agent {{agent_name}} may roll back a production deployment without waiting for review, and must file the change record within one hour after the fact. {#during-incident-agent-roll-back}

2. An emergency rollback must be announced in the incident channel before it is executed, not only after. {#emergency-rollback-announced-incident-channel}

3. Emergency authority granted during an incident expires when the incident is closed and does not carry over to unrelated changes. {#emergency-authority-granted-during-incident}
