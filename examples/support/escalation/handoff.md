---
title: Handoff
id: support.escalation.handoff
---

<!-- alaws:commentary -->

Rules for what a handoff from an agent to a human must include. The
[severity triage](alaws:support.escalation.severity_triage.ticket-triaged-within-sla-window) determines urgency, and the original
[severity classification](alaws:support.escalation.severity_triage.agents-downgrade-customer-reported-severity-without) must be preserved through the handoff
unless a human explicitly re-triages.

<!-- alaws:laws -->

1. When {{agent_name}} hands off ticket {{ticket_id}} to a human, the handoff note must include what was tried and why it wasn't sufficient. {#hands-off-ticket-human-handoff}

2. A ticket must not be handed off silently; the customer must be told a human is taking over. {#ticket-handed-off-silently-customer}

3. An escalated ticket must retain its original severity unless a human explicitly re-triages it. {#escalated-ticket-retain-original-severity}
