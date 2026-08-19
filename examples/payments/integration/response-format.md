---
title: Response Format
id: payments.integration.response_format
---

<!-- alaws:commentary -->

Rules for how an agent must respond when it authorizes a transaction or
decides a refund. A structured response is what makes a decision
auditable; a prose explanation is not. The required shape:

```json
{
  "decision": "approve" | "deny",
  "laws": ["<citation>", "..."],
  "reasoning": "<string>"
}
```

<!-- alaws:laws -->

1. When an agent authorizes, denies, or refunds a transaction, it must respond with structured JSON matching the schema in this section's commentary, not prose. {#agent-authorizes-denies-refunds-transaction}

2. Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision. {#every-citation-laws-field-one}

3. A "deny" decision must cite at least one law that justifies it. {#deny-decision-cite-least-one}
