---
title: Variables
id: support.integration.variables
---

<!-- alaws:commentary -->

This book's laws reference the following `{{variables}}` (docs/PLAN1.md
§17a); an application must supply all of them before rendering a law that
uses them:

* `customer_id` - the customer whose data or ticket is involved.
* `ticket_id` - the support ticket being handled.
* `agent_name` - the identity of the acting agent.

<!-- alaws:laws -->

1. Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently.
