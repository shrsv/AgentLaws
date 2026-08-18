# AgentLaws Examples

Three realistic, independently governed lawbooks, meant to exercise the
tool the way a real team would - not toy fixtures. Each is deeper and
larger than the minimal `fixtures/basic` lawbook used by tests, and each
threads `{{variable}}` placeholders through several laws so there's real
material to render for an agent prompt.

```text
examples/
├── engineering/    Engineering Governance      19 sections, 4 levels deep
├── payments/       Payments Authorization & Refunds
├── support/        Customer Support Governance
└── integration/    a runnable program: input -> prompt -> structured output -> audit trail
```

All three are discoverable and compile cleanly:

```text
$ alaws books list --root examples
examples/engineering  Engineering Governance
examples/payments  Payments Authorization & Refunds
examples/support  Customer Support Governance

$ alaws compile examples/engineering examples/payments examples/support --format html,json,pdf
compiled examples/engineering: 19 sections, 0 diagnostics -> examples/engineering/.alaws/build
compiled examples/payments: 9 sections, 0 diagnostics -> examples/payments/.alaws/build
compiled examples/support: 9 sections, 0 diagnostics -> examples/support/.alaws/build
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

## Assembling a full agent prompt: input, role, and structured output

Rendering laws (above) is only the middle of the loop. A real integration
also needs: runtime input, a role/task framing for the model, and a
response shape it can parse back into an auditable decision. AgentLaws
deliberately stops at "here are the applicable laws, with variables
substituted" (README "Using Laws from Go") - everything else here is the
*application's* responsibility, not the lawbook's. `examples/integration/`
is a complete, runnable, self-contained program (no LLM API key or network
access needed - it hardcodes a plausible model response so it's
deterministic) that does all of it for one concrete task, authorizing a
payment:

```bash
cd examples/integration && go run .
```

**1. Runtime input.** An ordinary Go struct - not every field becomes a
law `{{variable}}`:

```go
type TransactionRequest struct {
    TransactionID string
    Amount        float64
    Currency      string
    MerchantID    string
    AgentName     string
}
```

**2. Interpreting input as law variables.** A small, explicit mapping from
input fields to the `{{variable}}` names the *laws* actually use
(`amount`, `currency`, `merchant_id`, `agent_name` - not `transaction_id`,
which the laws never reference):

```go
vars := map[string]string{
    "amount":      fmt.Sprintf("%.2f", req.Amount),
    "currency":    req.Currency,
    "merchant_id": req.MerchantID,
    "agent_name":  req.AgentName,
}
rendered, _ := laws.Render(alaws.RenderOptions{Vars: vars, OnMissing: alaws.MissingError})
```

**3. The role.** Plain Go string-building, entirely outside AgentLaws -
the application decides what persona and task framing the model gets:

```go
role := fmt.Sprintf(`You are %s, a payments authorization agent. Decide whether to
approve or deny transaction %s (%.2f %s to %s). Ground your decision only
in the laws below, and cite the specific law numbers that informed it.

Respond with JSON only, in exactly this shape:
{"decision": "approve" | "deny", "laws": ["<citation>", ...], "reasoning": "<one paragraph>"}
`, req.AgentName, req.TransactionID, req.Amount, req.Currency, req.MerchantID)

prompt := role + "\nApplicable laws:\n\n" + rendered
```

Real, captured output of the assembled prompt:

```text
=== Assembled prompt ===
You are payments-authorizer, a payments authorization agent. Decide whether to
approve or deny transaction txn_8f2a91 (4200.00 USD to merchant_privet_drive_4). Ground your decision only
in the laws below, and cite the specific law numbers that informed it.

Respond with JSON only, in exactly this shape:
{"decision": "approve" | "deny", "laws": ["<citation>", ...], "reasoning": "<one paragraph>"}

Applicable laws:

1.1.1 A transaction above 4200.00 USD to merchant merchant_privet_drive_4 must pass step-up verification before it is authorized.
1.1.2 An agent must not increase a customer's transaction limit without an explicit, logged customer request.
1.1.3 Velocity limits (transactions per hour) must be enforced even when each individual transaction is within its own limit.
1.2.1 A transaction flagged by the fraud model must not be auto-approved by an agent, regardless of confidence score.
1.2.2 Agents must not disclose to a customer which specific fraud signal triggered a hold.
1.2.3 A false positive must be logged with enough detail to retrain the fraud model, not simply overridden and forgotten.
```

**4/5. Structured output, parsed and resolved back to source.** The model
is asked for JSON, not prose, specifically so the response can be
unmarshaled and its citations resolved deterministically - this is the
audit trail:

```go
type Decision struct {
    Decision  string   `json:"decision"`
    Laws      []string `json:"laws"`
    Reasoning string   `json:"reasoning"`
}

var decision Decision
json.Unmarshal([]byte(modelResponse), &decision)

for _, citation := range decision.Laws {
    law, _ := book.Resolve(citation)
    fmt.Printf("  %s  %s\n        source: %s:%d\n", law.Number, law.Text, law.Source.Path, law.Source.LineStart)
}
```

```text
=== Decision ===
DENY: The transaction exceeds the step-up verification threshold and no step-up verification was recorded, and it was independently flagged by the fraud model; per 1.2.1 a flagged transaction may not be auto-approved.

Cited laws, resolved to source:
  1.1.1  A transaction above {{amount}} {{currency}} to merchant {{merchant_id}} must pass step-up verification before it is authorized.
        source: ../payments/authorization/transaction-limits.md:12
  1.2.1  A transaction flagged by the fraud model must not be auto-approved by an agent, regardless of confidence score.
        source: ../payments/authorization/fraud-checks.md:12
```

(the source path is relative to `examples/integration/`, since that's where `go run .` above was invoked from - `book, _ := alaws.Load("../payments")`)

Notice `Resolve()` returns the law's *canonical* text, `{{amount}}` still
literal - that's the deterministic, signable source (docs/PLAN1.md §17a).
Only the earlier `Render()` call, for the prompt, substituted variables;
resolving a citation for an audit trail and rendering one for a prompt are
different operations, deliberately.

**The same `Decision{decision, laws, reasoning}` shape generalizes** to
the other two books - only the `decision` field's meaning changes:

```text
engineering:  {"decision": "approve" | "reject", "laws": [...], "reasoning": "..."}
              e.g. approving a deployment, citing engineering.operations.deployment laws

support:      {"decision": "resolve" | "escalate", "laws": [...], "reasoning": "..."}
              e.g. triaging a ticket, citing support.escalation.severity_triage laws
```

The shape (decision + cited laws + reasoning) is what makes any of these
audits mechanical; the vocabulary of `decision` is the only thing that's
domain-specific.

---

## The integration contract lives in the lawbook itself, not just a doc

The response format and variable list above aren't only documented
out-of-band (this README, `examples/integration/`) - each book states them
as an actual "Agent Integration" chapter, with citable laws, so they show
up in `alaws compile`/`export` output (HTML, PDF, Markdown, JSON) the same
as every other governance rule, because that's what they are:

```text
$ alaws list examples/payments | tail -6
3 Agent Integration (payments.integration)
3.1 Response Format (payments.integration.response_format)
  3.1.1 When an agent authorizes, denies, or refunds a transaction, it must respond with structured JSON, not prose, in exactly this shape: `{"decision": "approve" | "deny", "laws": ["<citation>", ...], "reasoning": "<string>"}`.
  3.1.2 Every citation in the `laws` field must be one of the laws actually supplied to the agent for that decision.
  3.1.3 A "deny" decision must cite at least one law that justifies it.
3.2 Variables (payments.integration.variables)
  3.2.1 Applications rendering this lawbook's laws for a prompt must supply a value for every variable referenced by the laws selected, or the render must fail rather than substitute a placeholder silently.
```

Every book has this chapter: `engineering.integration` (6), `payments.integration` (3), `support.integration` (3).
An agent that ignores the required response shape isn't just failing an
informal convention - it's violating `payments.integration.response_format`
citation `3.1.1`, the same as it would be violating any other law in the
book.

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

## Exporting everything, not just one book

`alaws compile` produces one set of artifacts per book. To hand someone
the whole governance program - all three books - as a single file,
`alaws export` compiles every book under a root and renders them into one
combined document, each book as its own part:

```text
$ alaws export examples --format html,pdf,md
exported 3 book(s) -> examples/.alaws/export
```

`examples/.alaws/export/lawbook.html` (and `.pdf`, `.md`) contains
Engineering Governance, Payments Authorization & Refunds, and Customer
Support Governance in one document, each under its own heading. `md`
(Markdown) is a supported format alongside `html`/`pdf`/`json` everywhere
formats are accepted - `alaws compile`, `alaws export`, and the web UI's
export buttons - useful when the destination is something that reads
Markdown natively (a wiki, a PR description, another Markdown-based tool)
rather than a browser or a printer. The web UI's book-list home page and
each book's own detail view both have "Export all"/"Export" buttons for
all three formats, backed by `GET /api/export` and `GET /api/book/export`.

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
