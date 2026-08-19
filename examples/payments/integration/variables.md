---
title: Variables
id: payments.integration.variables
---

<!-- alaws:commentary -->

This book's laws reference the following `{{variables}}` (docs/PLAN1.md
§17a); an application must supply all of them before rendering a law that
uses them:

* `amount` - the transaction or refund amount.
* `currency` - the currency the amount is denominated in.
* `merchant_id` - the merchant the transaction is with.
* `agent_name` - the identity of the acting agent.

<!-- alaws:laws -->

1. Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently. {#applications-rendering-lawbooks-laws-prompt}
