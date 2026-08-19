---
title: Authentication
id: engineering.security.authentication
---

<!-- alaws:commentary -->

Rules governing how an agent authenticates to internal and third-party
systems while performing a task.

<!-- alaws:laws -->

1. Agent {{agent_name}} must authenticate using short-lived, scoped credentials rather than long-lived API keys wherever the target system supports it. {#agent-authenticate-using-short-lived-scoped}

2. Agents must not share authentication tokens between unrelated tasks or sessions. {#agents-share-authentication-tokens-between}

3. A failed authentication attempt must be logged with the agent's identity and the resource it attempted to access. {#failed-authentication-attempt-logged-agents}
