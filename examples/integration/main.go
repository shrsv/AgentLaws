// Command integration is a runnable, self-contained demonstration of the
// part AgentLaws itself deliberately does NOT do: turning runtime input
// into a full agent prompt, and turning the agent's response back into an
// auditable decision. AgentLaws's job stops at "here are the applicable
// laws, with {{variables}} substituted" (README "Using Laws from Go"); an
// application still has to:
//
//  1. decide which laws apply to the task at hand,
//  2. map its own runtime input onto the {{variable}} names those laws use,
//  3. write a role/task preamble framing what the agent is being asked to
//     decide (AgentLaws has no opinion on this text - it's the
//     application's prompt, not the lawbook's),
//  4. tell the model what shape to respond in, so the response can be
//     parsed deterministically instead of scraped from prose, and
//  5. resolve whatever law citations come back to their exact source, so
//     "the model said so" becomes "clause 1.1.1 of transaction-limits.md,
//     as of this Git revision, said so" - the audit trail the whole
//     project exists for (README "Agent Citations").
//
// This example does all five steps for one concrete task - authorizing a
// payment transaction, using examples/payments - without calling a real
// LLM (step 6 below is a hardcoded stand-in response, so this program
// runs deterministically with no network access or API key).
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/shrsv/AgentLaws/pkg/alaws"
)

// TransactionRequest is the runtime input a payments service would
// actually have on hand - not every field here is a law {{variable}};
// transaction_id, for instance, is only used in the role/task framing
// below, never in the laws themselves.
type TransactionRequest struct {
	TransactionID string
	Amount        float64
	Currency      string
	MerchantID    string
	AgentName     string
}

// Decision is the structured shape we require the model to respond in.
// Asking for prose ("I approved this because...") makes an agent's output
// impossible to parse reliably; asking for this shape means the citations
// can be mechanically resolved back to source (step 5).
type Decision struct {
	Decision  string   `json:"decision"` // "approve" or "deny"
	Laws      []string `json:"laws"`     // citations, e.g. "1.1.1"
	Reasoning string   `json:"reasoning"`
}

func main() {
	req := TransactionRequest{
		TransactionID: "txn_8f2a91",
		Amount:        4200.00,
		Currency:      "USD",
		MerchantID:    "merchant_privet_drive_4",
		AgentName:     "payments-authorizer",
	}

	book, err := alaws.Load("../payments")
	if err != nil {
		log.Fatalf("load book: %v", err)
	}

	// Step 1: which laws apply. A real service would decide this from the
	// task type; here, authorizing a transaction clearly implicates both
	// of Authorization's child sections.
	laws, err := book.Laws(alaws.Selector{
		SectionIDs: []string{
			"payments.authorization.transaction_limits",
			"payments.authorization.fraud_checks",
		},
	})
	if err != nil {
		log.Fatalf("select laws: %v", err)
	}

	// Step 2: map runtime input onto the {{variable}} names the LAWS use
	// (docs/PLAN1.md §17a) - not every input field needs an entry here.
	vars := map[string]string{
		"amount":      fmt.Sprintf("%.2f", req.Amount),
		"currency":    req.Currency,
		"merchant_id": req.MerchantID,
		"agent_name":  req.AgentName,
	}
	rendered, err := laws.Render(alaws.RenderOptions{Vars: vars, OnMissing: alaws.MissingError})
	if err != nil {
		log.Fatalf("render laws: %v", err)
	}

	// Step 3+4: the application's own prompt framing - role, task, and
	// required response shape. AgentLaws has no involvement in this text;
	// it only produced `rendered` above.
	role := fmt.Sprintf(`You are %s, a payments authorization agent. Decide whether to
approve or deny transaction %s (%.2f %s to %s). Ground your decision only
in the laws below, and cite the specific law numbers that informed it.

Respond with JSON only, in exactly this shape:
{"decision": "approve" | "deny", "laws": ["<citation>", ...], "reasoning": "<one paragraph>"}
`, req.AgentName, req.TransactionID, req.Amount, req.Currency, req.MerchantID)

	prompt := role + "\nApplicable laws:\n\n" + rendered

	fmt.Println("=== Assembled prompt ===")
	fmt.Println(prompt)

	// Step 5 (stand-in): what a model would return for this prompt. A real
	// integration sends `prompt` to an LLM and reads its response here
	// instead of using a constant - this example hardcodes a plausible one
	// so it runs standalone.
	simulatedModelResponse := `{
  "decision": "deny",
  "laws": ["1.1.1", "1.2.1"],
  "reasoning": "The transaction exceeds the step-up verification threshold and no step-up verification was recorded, and it was independently flagged by the fraud model; per 1.2.1 a flagged transaction may not be auto-approved."
}`

	var decision Decision
	if err := json.Unmarshal([]byte(simulatedModelResponse), &decision); err != nil {
		log.Fatalf("parse model response: %v", err)
	}

	// Step 6: resolve every cited law back to its exact source - this is
	// the audit trail. "The model denied it" is not auditable; "the model
	// denied it, citing 1.1.1 (examples/payments/authorization/
	// transaction-limits.md) and 1.2.1 (.../fraud-checks.md)" is.
	fmt.Println("\n=== Decision ===")
	fmt.Printf("%s: %s\n\n", strings.ToUpper(decision.Decision), decision.Reasoning)
	fmt.Println("Cited laws, resolved to source:")
	for _, citation := range decision.Laws {
		law, err := book.Resolve(citation)
		if err != nil {
			log.Fatalf("resolve %s: %v", citation, err)
		}
		fmt.Printf("  %s  %s\n", law.Number, law.Text)
		fmt.Printf("        source: %s:%d\n", law.Source.Path, law.Source.LineStart)
	}

	fmt.Println("\nNote: Resolve() above returns the law's canonical text, with any")
	fmt.Println("{{variables}} still literal - that's the deterministic, signable source")
	fmt.Println("(docs/PLAN1.md §17a). Only the earlier Render() call for the prompt")
	fmt.Println("substituted them; resolving a citation for an audit trail is a separate")
	fmt.Println("concern from rendering one for a prompt, and intentionally so.")
}
