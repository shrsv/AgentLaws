---
title: Variables
id: engineering.integration.variables
---

<!-- alaws:commentary -->

This book's laws reference the following `{{variables}}` (docs/PLAN1.md
§17a); an application must supply all of them before rendering a law that
uses them:

* `agent_name` - the identity of the acting agent.
* `repo` - the repository being operated on.
* `reviewer` - a human reviewer's identifier, where a law requires one.
* `environment` - the deployment target (e.g. `staging`, `production`).
* `incident_id` - the active incident's identifier.
* `severity` - an incident's assigned severity level.

<!-- alaws:laws -->

1. Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently.
