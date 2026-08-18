# All Governance

## Engineering Governance

### 1 Principles

`engineering.principles`

These are the general principles that govern every agent working in this
codebase, regardless of task. More specific chapters (Security, Coding,
Operations, Incident Response) refine or add to these; none of them
override a principle stated here.

**1.1** Agents must prefer small, reviewable changes over large, sweeping rewrites.

**1.2** Agents must not merge code without human review unless the change is explicitly pre-authorized for autonomous merge.

**1.3** Agents must explain their reasoning when a decision is not obvious from the diff alone.

### 2 Security

`engineering.security`

This chapter covers how agents authenticate to systems, handle secrets, and
vet dependencies. It is organized into three subsections; this chapter
itself states no laws directly - see Authentication, Secrets, and
Dependencies below.

#### 2.1 Authentication

`engineering.security.authentication`

Rules governing how an agent authenticates to internal and third-party
systems while performing a task.

**2.1.1** Agent {{agent_name}} must authenticate using short-lived, scoped credentials rather than long-lived API keys wherever the target system supports it.

**2.1.2** Agents must not share authentication tokens between unrelated tasks or sessions.

**2.1.3** A failed authentication attempt must be logged with the agent's identity and the resource it attempted to access.

#### 2.2 Secrets

`engineering.security.secrets`

Rules for how agents handle credentials discovered in, or introduced into,
the repository.

**2.2.1** Credentials must never be committed to source control, including in commit messages or code comments.

**2.2.2** Agents must not print credentials into logs, error messages, or any output that may be persisted.

**2.2.3** Credentials discovered in source must be treated as compromised and rotated, not merely removed.

**2.2.4** Secrets required at runtime must be retrieved from the approved secret store, never hardcoded.

#### 2.3 Dependencies

`engineering.security.dependencies`

Rules for adding, upgrading, and evaluating third-party dependencies.

**2.3.1** Before adding a new dependency to {{repo}}, an agent must check it for known vulnerabilities using the approved scanner.

**2.3.2** Agents must not upgrade a dependency across a major version without flagging the change for human review.

**2.3.3** Dependencies with no maintenance activity in the last two years must be flagged as a risk, not silently relied upon.

### 3 Coding

`engineering.coding`

Rules for how agents make and submit code changes. See Code Review and
Testing below; this chapter itself states no laws directly.

#### 3.1 Code Review

`engineering.coding.review`

Rules for how a code change gets reviewed before it merges.

**3.1.1** Every change to {{repo}} must be reviewed by at least one human, {{reviewer}} or another qualified reviewer, before merging.

**3.1.2** An agent must not approve its own pull request.

**3.1.3** Review comments that request a change must be resolved or explicitly declined with a rationale before merge.

#### 3.2 Testing

`engineering.coding.testing`

Rules for what test coverage a change needs before it can be proposed.

**3.2.1** A change that modifies behavior must include or update an automated test that would fail without the change.

**3.2.2** Agents must not disable or skip a failing test to make a build pass without flagging it to a human.

**3.2.3** Test suites must be run locally or in CI before a change is proposed for review.

### 4 Operations

`engineering.operations`

Rules for deploying and operating production systems. See Deployment,
Monitoring, and Rollback below; this chapter itself states no laws
directly.

#### 4.1 Deployment

`engineering.operations.deployment`

Rules for pushing a change to a running environment.

**4.1.1** Agent {{agent_name}} must not deploy directly to the {{environment}} environment without an approved change record.

**4.1.2** Deployments to production must be preceded by a successful deployment to staging with the same artifact.

**4.1.3** A deployment must be reversible within one command or one documented procedure.

#### 4.2 Monitoring

`engineering.operations.monitoring`

Rules for alerting and anomaly handling once a service is live.

**4.2.1** A new production service must have baseline alerting configured before it receives real traffic.

**4.2.2** Agents must not silence an alert without recording the reason and an expiry for the silence.

**4.2.3** Anomalies detected by an agent must be reported even if the agent itself is not the cause.

#### 4.3 Rollback

`engineering.operations.rollback`

General rollback procedure: a deployment that causes an outage should be
rolled back to the last known-good artifact rather than fixed forward
under pressure. Emergency Procedures below covers the case where there is
no time to wait for normal review.

##### 4.3.1 Emergency Procedures

`engineering.operations.rollback.emergency`

This section exists three levels deep in the lawbook (Operations >
Rollback > Emergency Procedures), and its citations reflect that:
1.x.y.z-style numbers. It covers the narrow exception to the normal
rollback and deployment rules that applies only during a declared
incident.

**4.3.1.1** During incident {{incident_id}}, agent {{agent_name}} may roll back a production deployment without waiting for review, and must file the change record within one hour after the fact.

**4.3.1.2** An emergency rollback must be announced in the incident channel before it is executed, not only after.

**4.3.1.3** Emergency authority granted during an incident expires when the incident is closed and does not carry over to unrelated changes.

### 5 Incident Response

`engineering.incident_response`

Rules for classifying and communicating about production incidents. See
Severity Levels and Communication below; this chapter itself states no
laws directly.

#### 5.1 Severity Levels

`engineering.incident_response.severity_levels`

How an incident's severity is assigned and revised.

**5.1.1** An incident that causes customer-visible data loss must be classified Severity 1 regardless of the number of customers affected.

**5.1.2** Severity classification must be reassessed if new information changes the blast radius, not fixed at first report.

**5.1.3** Agents must not downgrade a severity level without a human confirming the downgrade.

#### 5.2 Communication

`engineering.incident_response.communication`

Rules for status updates and customer-facing messaging during and after an
incident.

**5.2.1** Incident {{incident_id}} at severity {{severity}} must have a status update posted at least once per hour until resolved.

**5.2.2** Customer-facing communication about an incident must be reviewed by a human before it is sent.

**5.2.3** Post-incident reviews must be published within five business days of resolution.

### 6 Agent Integration

`engineering.integration`

This chapter governs how a tool or agent must consume this lawbook and
respond, not what it must do to code or systems. See Response Format and
Variables below; this chapter itself states no laws directly.

#### 6.1 Response Format

`engineering.integration.response_format`

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

**6.1.1** When an agent makes a decision governed by this lawbook, it must respond with structured JSON matching the schema in this section's commentary, not prose.

**6.1.2** Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision; citing a law it was never given is itself a violation of this section.

**6.1.3** An "approve" or "reject" decision must cite at least one law, unless no law in this book applied to the task at hand.

#### 6.2 Variables

`engineering.integration.variables`

This book's laws reference the following `{{variables}}` (docs/PLAN1.md
§17a); an application must supply all of them before rendering a law that
uses them:

* `agent_name` - the identity of the acting agent.
* `repo` - the repository being operated on.
* `reviewer` - a human reviewer's identifier, where a law requires one.
* `environment` - the deployment target (e.g. `staging`, `production`).
* `incident_id` - the active incident's identifier.
* `severity` - an incident's assigned severity level.

**6.2.1** Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently.

## Payments Authorization & Refunds

### 1 Authorization

`payments.authorization`

Rules for authorizing a transaction before it settles. See Transaction
Limits and Fraud Checks below; this chapter itself states no laws
directly.

#### 1.1 Transaction Limits

`payments.authorization.transaction_limits`

Rules for per-transaction and velocity limits.

**1.1.1** A transaction above {{amount}} {{currency}} to merchant {{merchant_id}} must pass step-up verification before it is authorized.

**1.1.2** An agent must not increase a customer's transaction limit without an explicit, logged customer request.

**1.1.3** Velocity limits (transactions per hour) must be enforced even when each individual transaction is within its own limit.

#### 1.2 Fraud Checks

`payments.authorization.fraud_checks`

Rules for how an agent handles a transaction flagged by the fraud model.

**1.2.1** A transaction flagged by the fraud model must not be auto-approved by an agent, regardless of confidence score.

**1.2.2** Agents must not disclose to a customer which specific fraud signal triggered a hold.

**1.2.3** A false positive must be logged with enough detail to retrain the fraud model, not simply overridden and forgotten.

### 2 Refunds

`payments.refunds`

Rules for approving and communicating refunds. See Approval Thresholds and
Customer Communication below; this chapter itself states no laws directly.

#### 2.1 Approval Thresholds

`payments.refunds.approval_thresholds`

Rules for how much of a refund an agent can approve on its own authority.

**2.1.1** Agent {{agent_name}} may approve a refund up to {{amount}} {{currency}} without additional sign-off.

**2.1.2** A refund above the agent's approval threshold must be routed to a human approver with the original transaction attached.

**2.1.3** Refunds must not be split into smaller amounts to stay under an approval threshold.

#### 2.2 Customer Communication

`payments.refunds.customer_communication`

Rules for what a customer must be told about a refund.

**2.2.1** A customer must be notified when their refund is approved and again when funds are received, not only once.

**2.2.2** Agents must not promise a specific refund timeline that the payment processor cannot guarantee.

**2.2.3** Refund denials must include the specific reason, not a generic rejection message.

### 3 Agent Integration

`payments.integration`

This chapter governs how a tool or agent must consume this lawbook and
respond, not what it must do with a transaction or refund. See Response
Format and Variables below; this chapter itself states no laws directly.

#### 3.1 Response Format

`payments.integration.response_format`

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

**3.1.1** When an agent authorizes, denies, or refunds a transaction, it must respond with structured JSON matching the schema in this section's commentary, not prose.

**3.1.2** Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision.

**3.1.3** A "deny" decision must cite at least one law that justifies it.

#### 3.2 Variables

`payments.integration.variables`

This book's laws reference the following `{{variables}}` (docs/PLAN1.md
§17a); an application must supply all of them before rendering a law that
uses them:

* `amount` - the transaction or refund amount.
* `currency` - the currency the amount is denominated in.
* `merchant_id` - the merchant the transaction is with.
* `agent_name` - the identity of the acting agent.

**3.2.1** Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently.

## Customer Support Governance

### 1 Customer Data

`support.customer_data`

Rules for handling personal data encountered while resolving a ticket. See
PII Handling and Retention below; this chapter itself states no laws
directly.

#### 1.1 PII Handling

`support.customer_data.pii_handling`

Rules for keeping personal information out of places it shouldn't end up.

**1.1.1** Agent {{agent_name}} must not paste customer {{customer_id}}'s personal information into a ticket visible outside the support system.

**1.1.2** Full card numbers, SSNs, and passwords must never appear in a support ticket, chat transcript, or agent note.

**1.1.3** Access to a customer's account must be logged with the reason for access, not just the fact of access.

#### 1.2 Retention

`support.customer_data.retention`

Rules for how long customer data is kept and how it is deleted on
request.

**1.2.1** A closed ticket containing PII must be redacted or deleted according to the retention schedule, not kept indefinitely by default.

**1.2.2** A customer's data deletion request must be honored within the legally required window, tracked to completion.

**1.2.3** Agents must not export customer data to a personal device or unapproved tool for any reason.

### 2 Escalation

`support.escalation`

Rules for triaging and handing off a ticket to a human. See Severity
Triage and Handoff below; this chapter itself states no laws directly.

#### 2.1 Severity Triage

`support.escalation.severity_triage`

Rules for assigning and revising a ticket's severity.

**2.1.1** Ticket {{ticket_id}} must be triaged within the SLA window appropriate to its stated severity, not first-in-first-out regardless of severity.

**2.1.2** A ticket reporting potential account takeover must be escalated immediately, bypassing the normal queue.

**2.1.3** Agents must not downgrade a customer-reported severity without documenting why.

#### 2.2 Handoff

`support.escalation.handoff`

Rules for what a handoff from an agent to a human must include.

**2.2.1** When {{agent_name}} hands off ticket {{ticket_id}} to a human, the handoff note must include what was tried and why it wasn't sufficient.

**2.2.2** A ticket must not be handed off silently; the customer must be told a human is taking over.

**2.2.3** An escalated ticket must retain its original severity unless a human explicitly re-triages it.

### 3 Agent Integration

`support.integration`

This chapter governs how a tool or agent must consume this lawbook and
respond, not what it must do with a ticket. See Response Format and
Variables below; this chapter itself states no laws directly.

#### 3.1 Response Format

`support.integration.response_format`

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

**3.1.1** When an agent triages or closes a ticket, it must respond with structured JSON matching the schema in this section's commentary, not prose.

**3.1.2** Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision.

**3.1.3** An "escalate" decision must cite at least one law that justifies it.

#### 3.2 Variables

`support.integration.variables`

This book's laws reference the following `{{variables}}` (docs/PLAN1.md
§17a); an application must supply all of them before rendering a law that
uses them:

* `customer_id` - the customer whose data or ticket is involved.
* `ticket_id` - the support ticket being handled.
* `agent_name` - the identity of the acting agent.

**3.2.1** Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently.

