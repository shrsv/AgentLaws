# Engineering Governance

## 1 Principles

`engineering.principles`

These are the general principles that govern every agent working in this
codebase, regardless of task. More specific chapters (Security, Coding,
Operations, Incident Response) refine or add to these; none of them
override a principle stated here.

For example, the principle of small reviewable changes is complemented by
[review requirements](#engineering.coding.review.every-change-reviewed-least-one)
and [testing obligations](#engineering.coding.testing.change-modifies-behavior-include-update).
Agents must also follow [secrets handling](#engineering.security.secrets.credentials-never-committed-source-control) when working with credentials.

<a id="engineering.principles.agents-prefer-small-reviewable-changes"></a>
**1.1** Agents must prefer small, reviewable changes over large, sweeping rewrites.

<a id="engineering.principles.agents-merge-code-without-human"></a>
**1.2** Agents must not merge code without human review unless the change is explicitly pre-authorized for autonomous merge.

<a id="engineering.principles.agents-explain-their-reasoning-decision"></a>
**1.3** Agents must explain their reasoning when a decision is not obvious from the diff alone.

## 2 Security

`engineering.security`

This chapter covers how agents authenticate to systems, handle secrets, and
vet dependencies. It is organized into three subsections; this chapter
itself states no laws directly - see
[Authentication](#engineering.security.authentication),
[Secrets](#engineering.security.secrets), and
[Dependencies](#engineering.security.dependencies) below.

### 2.1 Authentication

`engineering.security.authentication`

Rules governing how an agent authenticates to internal and third-party
systems while performing a task.

<a id="engineering.security.authentication.agent-authenticate-using-short-lived-scoped"></a>
**2.1.1** Agent {{agent_name}} must authenticate using short-lived, scoped credentials rather than long-lived API keys wherever the target system supports it.

<a id="engineering.security.authentication.agents-share-authentication-tokens-between"></a>
**2.1.2** Agents must not share authentication tokens between unrelated tasks or sessions.

<a id="engineering.security.authentication.failed-authentication-attempt-logged-agents"></a>
**2.1.3** A failed authentication attempt must be logged with the agent's identity and the resource it attempted to access.

### 2.2 Secrets

`engineering.security.secrets`

Rules for how agents handle credentials discovered in, or introduced into,
the repository. These rules work alongside
[authentication requirements](#engineering.security.authentication.agent-authenticate-using-short-lived-scoped) -
a leaked credential is useless if auth is short-lived and scoped.

<a id="engineering.security.secrets.credentials-never-committed-source-control"></a>
**2.2.1** Credentials must never be committed to source control, including in commit messages or code comments.

<a id="engineering.security.secrets.agents-print-credentials-into-logs"></a>
**2.2.2** Agents must not print credentials into logs, error messages, or any output that may be persisted.

<a id="engineering.security.secrets.credentials-discovered-source-treated-compromised"></a>
**2.2.3** Credentials discovered in source must be treated as compromised and rotated, not merely removed.

<a id="engineering.security.secrets.secrets-required-runtime-retrieved-approved"></a>
**2.2.4** Secrets required at runtime must be retrieved from the approved secret store, never hardcoded.

### 2.3 Dependencies

`engineering.security.dependencies`

Rules for adding, upgrading, and evaluating third-party dependencies.

<a id="engineering.security.dependencies.before-adding-new-dependency-agent"></a>
**2.3.1** Before adding a new dependency to {{repo}}, an agent must check it for known vulnerabilities using the approved scanner.

<a id="engineering.security.dependencies.agents-upgrade-dependency-across-major"></a>
**2.3.2** Agents must not upgrade a dependency across a major version without flagging the change for human review.

<a id="engineering.security.dependencies.dependencies-maintenance-activity-last-two"></a>
**2.3.3** Dependencies with no maintenance activity in the last two years must be flagged as a risk, not silently relied upon.

## 3 Coding

`engineering.coding`

Rules for how agents make and submit code changes. See Code Review and
Testing below; this chapter itself states no laws directly.

### 3.1 Code Review

`engineering.coding.review`

Rules for how a code change gets reviewed before it merges. A reviewer
should verify that [testing obligations](#engineering.coding.testing.change-modifies-behavior-include-update) have been met and that
[secrets are not introduced](#engineering.security.secrets.credentials-never-committed-source-control) into the change.

<a id="engineering.coding.review.every-change-reviewed-least-one"></a>
**3.1.1** Every change to {{repo}} must be reviewed by at least one human, {{reviewer}} or another qualified reviewer, before merging.

<a id="engineering.coding.review.agent-approve-own-pull-request"></a>
**3.1.2** An agent must not approve its own pull request.

<a id="engineering.coding.review.review-comments-request-change-resolved"></a>
**3.1.3** Review comments that request a change must be resolved or explicitly declined with a rationale before merge.

### 3.2 Testing

`engineering.coding.testing`

Rules for what test coverage a change needs before it can be proposed.
Tests are validated during [code review](#engineering.coding.review.every-change-reviewed-least-one), and failing tests must not be hidden
to circumvent the [review process](#engineering.coding.review.review-comments-request-change-resolved).

<a id="engineering.coding.testing.change-modifies-behavior-include-update"></a>
**3.2.1** A change that modifies behavior must include or update an automated test that would fail without the change.

<a id="engineering.coding.testing.agents-disable-skip-failing-test"></a>
**3.2.2** Agents must not disable or skip a failing test to make a build pass without flagging it to a human.

<a id="engineering.coding.testing.test-suites-run-locally-ci"></a>
**3.2.3** Test suites must be run locally or in CI before a change is proposed for review.

## 4 Operations

`engineering.operations`

Rules for deploying and operating production systems. See Deployment,
Monitoring, and Rollback below; this chapter itself states no laws
directly.

### 4.1 Deployment

`engineering.operations.deployment`

Rules for pushing a change to a running environment. Every deployment
must have a rollback plan (see [Rollback](#engineering.operations.rollback)) before it proceeds, and
[monitoring](#engineering.operations.monitoring) must be in place to detect failures quickly.

<a id="engineering.operations.deployment.agent-deploy-directly-environment-without"></a>
**4.1.1** Agent {{agent_name}} must not deploy directly to the {{environment}} environment without an approved change record.

<a id="engineering.operations.deployment.deployments-production-preceded-successful-deployment"></a>
**4.1.2** Deployments to production must be preceded by a successful deployment to staging with the same artifact.

<a id="engineering.operations.deployment.deployment-reversible-within-one-command"></a>
**4.1.3** A deployment must be reversible within one command or one documented procedure.

### 4.2 Monitoring

`engineering.operations.monitoring`

Rules for alerting and anomaly handling once a service is live.

<a id="engineering.operations.monitoring.new-production-service-baseline-alerting"></a>
**4.2.1** A new production service must have baseline alerting configured before it receives real traffic.

<a id="engineering.operations.monitoring.agents-silence-alert-without-recording"></a>
**4.2.2** Agents must not silence an alert without recording the reason and an expiry for the silence.

<a id="engineering.operations.monitoring.anomalies-detected-agent-reported-even"></a>
**4.2.3** Anomalies detected by an agent must be reported even if the agent itself is not the cause.

### 4.3 Rollback

`engineering.operations.rollback`

General rollback procedure: a deployment that causes an outage should be
rolled back to the last known-good artifact rather than fixed forward
under pressure. Rollback applies after a
[staging-to-production deployment](#engineering.operations.deployment.deployments-production-preceded-successful-deployment) fails, and the
[incident severity](#engineering.incident_response.severity_levels.incident-causes-customer-visible-data-loss) determines how quickly a rollback must execute.

#### 4.3.1 Emergency Procedures

`engineering.operations.rollback.emergency`

This section exists three levels deep in the lawbook (Operations >
Rollback > Emergency Procedures), and its citations reflect that:
1.x.y.z-style numbers. It covers the narrow exception to the normal
rollback and deployment rules that applies only during a declared
incident.

<a id="engineering.operations.rollback.emergency.during-incident-agent-roll-back"></a>
**4.3.1.1** During incident {{incident_id}}, agent {{agent_name}} may roll back a production deployment without waiting for review, and must file the change record within one hour after the fact.

<a id="engineering.operations.rollback.emergency.emergency-rollback-announced-incident-channel"></a>
**4.3.1.2** An emergency rollback must be announced in the incident channel before it is executed, not only after.

<a id="engineering.operations.rollback.emergency.emergency-authority-granted-during-incident"></a>
**4.3.1.3** Emergency authority granted during an incident expires when the incident is closed and does not carry over to unrelated changes.

## 5 Incident Response

`engineering.incident_response`

Rules for classifying and communicating about production incidents. See
Severity Levels and Communication below; this chapter itself states no
laws directly.

### 5.1 Severity Levels

`engineering.incident_response.severity_levels`

How an incident's severity is assigned and revised. Severity directly
governs [communication cadence](#engineering.incident_response.communication.incident-severity-status-update-posted) and determines whether
automated [rollback](#engineering.operations.rollback) should be triggered.

<a id="engineering.incident_response.severity_levels.incident-causes-customer-visible-data-loss"></a>
**5.1.1** An incident that causes customer-visible data loss must be classified Severity 1 regardless of the number of customers affected.

<a id="engineering.incident_response.severity_levels.severity-classification-reassessed-new-information"></a>
**5.1.2** Severity classification must be reassessed if new information changes the blast radius, not fixed at first report.

<a id="engineering.incident_response.severity_levels.agents-downgrade-severity-level-without"></a>
**5.1.3** Agents must not downgrade a severity level without a human confirming the downgrade.

### 5.2 Communication

`engineering.incident_response.communication`

Rules for status updates and customer-facing messaging during and after an
incident. Communication frequency is driven by
[severity level](#engineering.incident_response.severity_levels.incident-causes-customer-visible-data-loss), and post-incident reviews must reference the
original [severity classification](#engineering.incident_response.severity_levels.severity-classification-reassessed-new-information).

<a id="engineering.incident_response.communication.incident-severity-status-update-posted"></a>
**5.2.1** Incident {{incident_id}} at severity {{severity}} must have a status update posted at least once per hour until resolved.

<a id="engineering.incident_response.communication.customer-facing-communication-about-incident-reviewed"></a>
**5.2.2** Customer-facing communication about an incident must be reviewed by a human before it is sent.

<a id="engineering.incident_response.communication.post-incident-reviews-published-within-five"></a>
**5.2.3** Post-incident reviews must be published within five business days of resolution.

## 6 Agent Integration

`engineering.integration`

This chapter governs how a tool or agent must consume this lawbook and
respond, not what it must do to code or systems. See Response Format and
Variables below; this chapter itself states no laws directly.

### 6.1 Response Format

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

<a id="engineering.integration.response_format.agent-makes-decision-governed-lawbook"></a>
**6.1.1** When an agent makes a decision governed by this lawbook, it must respond with structured JSON matching the schema in this section's commentary, not prose.
   ```json
   {
     "decision": "approve",
     "laws": ["4.1.2", "4.1.3"],
     "reasoning": "The deployment passes all checks and the rollback path is ready."
   }
   ```

<a id="engineering.integration.response_format.every-citation-laws-field-one"></a>
**6.1.2** Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision; citing a law it was never given is itself a violation of this section.

<a id="engineering.integration.response_format.approve-reject-decision-cite-least"></a>
**6.1.3** An "approve" or "reject" decision must cite at least one law, unless no law in this book applied to the task at hand.

### 6.2 Variables

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

<a id="engineering.integration.variables.applications-rendering-lawbooks-laws-prompt"></a>
**6.2.1** Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently.

---

*revision `08cb50cdac66 (dirty)` · compiled 2026-08-19T19:29:04Z · by Shrijith Venkatramana · alaws v0.1.0-30-g08cb50c-dirty (built 2026-08-19T19:18:11Z)*

