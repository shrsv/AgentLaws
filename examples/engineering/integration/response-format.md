---
title: Response Format
id: engineering.integration.response_format
---

<!-- alaws:commentary -->

Rules for how an agent must respond when it makes a decision governed by
this lawbook - approving or rejecting a deployment, a pull request, or an
emergency rollback. A structured response is what makes a decision
auditable; a prose explanation is not. The required shape:

```json
{
  "decision": "approve" | "reject",
  "laws": ["<citation>", "..."],
  "reasoning": "<string>"
}
```

<!-- alaws:laws -->

1. When an agent makes a decision governed by this lawbook, it must respond with structured JSON matching the schema in this section's commentary, not prose.

2. Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision; citing a law it was never given is itself a violation of this section.

3. An "approve" or "reject" decision must cite at least one law, unless no law in this book applied to the task at hand.
