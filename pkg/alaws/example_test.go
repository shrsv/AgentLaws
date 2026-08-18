package alaws_test

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/shrsv/AgentLaws/pkg/alaws"
)

// This example demonstrates loading a lawbook, selecting laws by section,
// and rendering them as prompt-ready text with variable substitution.
func Example() {
	book, err := alaws.Load("examples/payments")
	if err != nil {
		log.Fatal(err)
	}

	laws, err := book.Laws(alaws.Selector{
		SectionIDs: []string{"payments.refunds.approval_thresholds"},
	})
	if err != nil {
		log.Fatal(err)
	}

	rendered, err := laws.Render(alaws.RenderOptions{
		Vars: map[string]string{
			"agent_name": "payments-bot",
			"amount":     "500",
			"currency":   "USD",
		},
		OnMissing: alaws.MissingError,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(rendered)
}

// This example demonstrates compiling a lawbook and inspecting diagnostics.
// Compile always returns a *Book even when the lawbook has errors, so a
// caller can show everything wrong rather than just the first problem.
func ExampleCompile() {
	book, err := alaws.Compile("examples/engineering")
	if err != nil {
		// Book is still usable — Diagnostics() has the full list.
		fmt.Printf("compile error: %v\n", err)
	}

	for _, d := range book.Diagnostics() {
		fmt.Printf("[%s] %s: %s\n", d.Severity, d.Code, d.Message)
	}

	lb := book.Lawbook()
	fmt.Printf("title: %s\nsections: %d\n", lb.Metadata.Title, len(lb.Sections))
}

// This example demonstrates resolving a canonical citation to its law,
// section, and source location — the core of the audit trail.
func ExampleBook_Resolve() {
	book, err := alaws.Load("examples/engineering")
	if err != nil {
		log.Fatal(err)
	}

	law, err := book.Resolve("2.5.1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Number:  %s\n", law.Number)
	fmt.Printf("Text:    %s\n", law.Text)
	fmt.Printf("Section: %s\n", law.SectionID)
	fmt.Printf("Source:  %s:%d\n", law.Source.Path, law.Source.LineStart)
}

// This example demonstrates selecting all laws from a book and rendering
// them into a single prompt-ready block — useful when an agent needs the
// full governance context.
func ExampleBook_Laws_all() {
	book, err := alaws.Load("examples/support")
	if err != nil {
		log.Fatal(err)
	}

	laws, err := book.Laws(alaws.Selector{All: true})
	if err != nil {
		log.Fatal(err)
	}

	rendered, err := laws.Render(alaws.RenderOptions{
		Vars:      map[string]string{"agent_name": "support-bot"},
		OnMissing: alaws.MissingKeepPlaceholder,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(rendered)
}

// This example demonstrates selecting laws by citation — for when an agent
// cited specific laws and you need to fetch just those.
func ExampleBook_Laws_citations() {
	book, err := alaws.Load("examples/payments")
	if err != nil {
		log.Fatal(err)
	}

	laws, err := book.Laws(alaws.Selector{
		Citations: []string{"1.1.1", "1.2.1"},
	})
	if err != nil {
		log.Fatal(err)
	}

	rendered, _ := laws.Render(alaws.RenderOptions{
		Vars: map[string]string{
			"amount":      "4200.00",
			"currency":    "USD",
			"merchant_id": "merchant_acme",
			"agent_name":  "payments-bot",
		},
	})
	fmt.Println(rendered)
}

// This example demonstrates the missing-variable policies: error (default),
// keep placeholder, and empty string.
func ExampleLawSet_Render_missingPolicy() {
	book, err := alaws.Load("examples/engineering")
	if err != nil {
		log.Fatal(err)
	}

	laws, _ := book.Laws(alaws.Selector{
		SectionIDs: []string{"engineering.operations.rollback.emergency"},
	})

	// Default: fail if a variable has no value.
	_, err = laws.Render(alaws.RenderOptions{
		Vars:      map[string]string{"agent_name": "ops-bot"}, // missing incident_id
		OnMissing: alaws.MissingError,
	})
	if err != nil {
		fmt.Printf("MissingError: %v\n", err)
	}

	// Keep: leave {{incident_id}} as-is in the output.
	rendered, _ := laws.Render(alaws.RenderOptions{
		Vars:      map[string]string{"agent_name": "ops-bot"},
		OnMissing: alaws.MissingKeepPlaceholder,
	})
	fmt.Println(rendered)
}

// This example demonstrates discovering all lawbooks under a directory.
func ExampleDiscover() {
	books, err := alaws.Discover("examples")
	if err != nil {
		log.Fatal(err)
	}

	for _, b := range books {
		fmt.Printf("%-30s %s\n", b.Path, b.Title)
	}
}

// This example demonstrates writing compiled artifacts (HTML, JSON, PDF,
// Markdown) to a build directory — the same thing `alaws compile` does.
func ExampleBook_WriteArtifacts() {
	book, err := alaws.Compile("examples/engineering")
	if err != nil {
		log.Fatal(err)
	}

	if err := book.WriteArtifacts(".alaws/build", "html,json"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("artifacts written to .alaws/build/")
}

// This example demonstrates rendering a compiled book as HTML to any
// writer — useful for embedding in a server or writing to a buffer.
func ExampleBook_RenderHTML() {
	book, err := alaws.Load("examples/engineering")
	if err != nil {
		log.Fatal(err)
	}

	var buf strings.Builder
	if err := book.RenderHTML(&buf); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("HTML output: %d bytes\n", buf.Len())
}

// This example demonstrates the full compile-all-books workflow: discover
// every book under a root, compile each, and render a combined document.
func ExampleCompileAll() {
	books, err := alaws.CompileAll("examples")
	if err != nil {
		fmt.Printf("some books had errors: %v\n", err)
	}

	fmt.Printf("compiled %d book(s)\n", len(books))

	// Write a single combined HTML file covering all books.
	f, _ := os.Create(".alaws/export/all.html")
	defer f.Close()
	if err := alaws.RenderCombinedHTML(f, "All Governance", books); err != nil {
		log.Fatal(err)
	}
}

// This example demonstrates looking up a section by its stable ID.
func ExampleBook_Section() {
	book, err := alaws.Load("examples/engineering")
	if err != nil {
		log.Fatal(err)
	}

	sec, err := book.Section("engineering.security.secrets")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID:      %s\n", sec.ID)
	fmt.Printf("Title:   %s\n", sec.Title)
	fmt.Printf("Level:   %d\n", sec.Level)
	fmt.Printf("Laws:    %d\n", len(sec.Laws))
}
