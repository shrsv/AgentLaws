---
title: Response Format
id: support.integration.response_format
---

<!-- alaws:commentary -->

Rules for how an agent must respond when it triages or resolves a ticket.
A structured response is what makes a decision auditable; a prose
explanation is not. The required shape:

```json
{
  "decision": "resolve" | "escalate",
  "laws": ["<citation>", "..."],
  "reasoning": "<string>"
}
```

<!-- alaws:laws -->

1. When an agent triages or closes a ticket, it must respond with structured JSON matching the schema in this section's commentary, not prose. {#agent-triages-closes-ticket-respond}

2. Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision. {#every-citation-laws-field-one}

3. An "escalate" decision must cite at least one law that justifies it. {#escalate-decision-cite-least-one}
