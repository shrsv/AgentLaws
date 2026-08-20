package alaws

import (
	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/resolver"
	"github.com/shrsv/AgentLaws/internal/template"
)

// Prompts returns every compiled PromptTemplate in the book, in
// promptTemplates order.
func (b *Book) Prompts() []model.PromptTemplate {
	return b.lawbook.Prompts
}

// Prompt resolves a prompt ID to a Prompt, which wraps the compiled
// PromptTemplate and adds a Render method.
func (b *Book) Prompt(id string) (Prompt, error) {
	pt, err := resolver.ResolvePrompt(b.lawbook, id)
	if err != nil {
		return Prompt{}, err
	}
	return Prompt{pt}, nil
}

// Prompt wraps a compiled PromptTemplate and adds prompt-specific methods.
type Prompt struct {
	model.PromptTemplate
}

// PromptRenderOptions configures Prompt.Render.
type PromptRenderOptions struct {
	Vars      map[string]string
	OnMissing MissingPolicy
}

// Render renders the prompt template, substituting {{variable}} placeholders
// per opts. The compiled Template field (with {{ref:x}} directives already
// expanded at compile time) is passed to internal/template.Render unchanged.
func (p Prompt) Render(opts PromptRenderOptions) (string, error) {
	return template.Render(p.Template, opts.Vars, opts.OnMissing)
}
