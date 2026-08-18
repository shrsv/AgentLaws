# AgentLaws Examples

Three realistic, independently governed lawbooks, meant to exercise the
tool the way a real team would - not toy fixtures. Each is deeper and
larger than the minimal `fixtures/basic` lawbook used by tests, and each
threads `{{variable}}` placeholders through several laws so there's real
material to render for an agent prompt.

```text
examples/
├── engineering/   Engineering Governance      16 sections, 4 levels deep
├── payments/      Payments Authorization & Refunds
└── support/       Customer Support Governance
```

All three are discoverable and compile cleanly:

```text
$ alaws books list --root examples
examples/engineering  Engineering Governance
examples/payments  Payments Authorization & Refunds
examples/support  Customer Support Governance

$ alaws compile examples/engineering examples/payments examples/support --format html,json,pdf
compiled examples/engineering: 16 sections, 0 diagnostics -> examples/engineering/.alaws/build
compiled examples/payments: 6 sections, 0 diagnostics -> examples/payments/.alaws/build
compiled examples/support: 6 sections, 0 diagnostics -> examples/support/.alaws/build
```

---

## Depth: level comes from folder depth, not folder names

`engineering/` goes four levels deep: `Operations` (a chapter) contains
`Rollback` (a section), which itself contains `Emergency Procedures` - a
section nested inside a section. None of these files has an explicit
`level:` in its frontmatter; level defaults to `1 + <path separators in
the file's ordering entry>`, so `operations/rollback/emergency.md`
(two separators) is level 3 without anyone writing that down:

```text
$ alaws books show examples/engineering
Engineering Governance  (examples/engineering)
  level 1  engineering.principles  (principles.md)
  level 1  engineering.security  (security.md)
  level 2  engineering.security.authentication  (security/authentication.md)
  ...
  level 1  engineering.operations  (operations.md)
  level 2  engineering.operations.deployment  (operations/deployment.md)
  level 2  engineering.operations.monitoring  (operations/monitoring.md)
  level 2  engineering.operations.rollback  (operations/rollback.md)
  level 3  engineering.operations.rollback.emergency  (operations/rollback/emergency.md)
  ...
```

which compiles to citation numbers that reflect the same four levels of
nesting - a law inside Emergency Procedures is `4.3.1.<N>`:

```text
$ alaws resolve 4.3.1.1 --root examples/engineering
4.3.1.1 During incident {{incident_id}}, agent {{agent_name}} may roll back a production deployment without waiting for review, and must file the change record within one hour after the fact.
  section: engineering.operations.rollback.emergency
  source:  examples/engineering/operations/rollback/emergency.md:16-17
```

Only where a section's intended depth *doesn't* match where its file
happens to live would you write `level:` explicitly - that's the
exception case, not the everyday one. `alaws section create --level N`
writes that override automatically when it's needed and omits it
otherwise; see `internal/cli/helpers.go`'s `levelOverride`.

Folder *names* still carry no meaning of their own - `operations/rollback/`
could just as well be named anything; only the nesting depth matters.

---

## Variables: rendering for an agent prompt

Laws that need a runtime value use `{{variable}}` placeholders. They stay
literal in the compiled, signed lawbook (see the `resolve` output above -
`{{incident_id}}` and `{{agent_name}}` are untouched) and are only
substituted when an application renders a prompt:

```text
$ alaws render --book examples/payments --section payments.refunds.approval_thresholds \
    --var agent_name=ci-bot --var amount=500 --var currency=USD
2.1.1 Agent ci-bot may approve a refund up to 500 USD without additional sign-off.
2.1.2 A refund above the agent's approval threshold must be routed to a human approver with the original transaction attached.
2.1.3 Refunds must not be split into smaller amounts to stay under an approval threshold.
```

A single law, rendered as JSON (what an application would actually call):

```text
$ alaws render --book examples/engineering --law 4.3.1.1 \
    --var incident_id=INC-482 --var agent_name=ops-bot --json
"4.3.1.1 During incident INC-482, agent ops-bot may roll back a production deployment without waiting for review, and must file the change record within one hour after the fact."
```

Leaving a variable unset is an error by default - a prompt should never
silently ship a raw `{{...}}` to a model. `--on-missing keep` opts out of
that, e.g. while iterating on wording before the calling application is
ready to supply every variable:

```text
$ alaws render --book examples/support --section support.customer_data.pii_handling \
    --var agent_name=support-bot --on-missing keep
1.1.1 Agent support-bot must not paste customer {{customer_id}}'s personal information into a ticket visible outside the support system.
1.1.2 Full card numbers, SSNs, and passwords must never appear in a support ticket, chat transcript, or agent note.
1.1.3 Access to a customer's account must be logged with the reason for access, not just the fact of access.
```

Variables used across the three books: `agent_name`, `repo`, `reviewer`,
`environment`, `incident_id`, `severity` (engineering); `amount`,
`currency`, `merchant_id`, `agent_name` (payments); `customer_id`,
`ticket_id`, `agent_name` (support).

---

## JSON output

Every read command takes `--json` for machine consumption - this is the
form an application or another CLI would actually parse, not the
human-readable summary:

```text
$ alaws list examples/payments --json
{
  "Metadata": {
    "Title": "Payments Authorization & Refunds",
    "Ordering": [ "authorization.md", "authorization/transaction-limits.md", ... ]
  },
  "Sections": [
    {
      "ID": "payments.authorization",
      "Number": "1",
      "Title": "Authorization",
      "Level": 1,
      "ParentID": "",
      "Source": { "Path": "examples/payments/authorization.md", "LineStart": 1, "LineEnd": 13 },
      "Commentary": "Rules for authorizing a transaction before it settles. ...",
      "Laws": null
    },
    ...
  ]
}
```

(`payments.authorization` itself has `"Laws": null` - it's a chapter that
only organizes Transaction Limits and Fraud Checks underneath it; see
"Chapters with laws of their own" below for why that split is required,
not just stylistic.)

`alaws compile --format json` writes the same canonical shape to
`.alaws/build/lawbook.json` as a build artifact.

---

## Chapters with laws of their own are rejected

A section can have child sections, or laws of its own, but not both - if
it had both, a law and a subsection could end up sharing one citation
number (e.g. both numbered `2.1`). Every chapter in these three books
(`Security`, `Coding`, `Operations`, `Rollback`, `Authorization`,
`Refunds`, `Customer Data`, `Escalation`, ...) therefore states no laws
directly; its commentary explains the chapter and its child sections carry
the actual rules. `alaws validate` catches the mistake if you get this
wrong, along with an empty-laws warning for a leaf section that has
neither laws nor children (a real captured example, from a throwaway
scratch book - not one of the three books above):

```text
$ alaws validate ./demo
./demo: error: ambiguous-numbering: demo.security: has both child sections and 1 law(s) of its own, which produces ambiguous citations; move these laws into a child section
./demo: warning: missing-laws: demo.security.secrets: laws region has no numbered clauses
./demo: warning: missing-laws: demo.empty: laws region has no numbered clauses
./demo: 1 error(s) found
alaws: validation failed for: ./demo
```

---

## Rebuilding an example from scratch, with the CLI

The `payments` book above was built the same way a team would build one -
here's `refunds` end to end:

```bash
alaws books create ./payments --title "Payments Authorization & Refunds"
alaws chapter create ./payments refunds.md --title Refunds --id payments.refunds
alaws section create ./payments refunds/approval-thresholds.md \
  --parent payments.refunds --title "Approval Thresholds" --id payments.refunds.approval_thresholds
alaws law add ./payments payments.refunds.approval_thresholds \
  "Agent {{agent_name}} may approve a refund up to {{amount}} {{currency}} without additional sign-off."
alaws compile ./payments
```
