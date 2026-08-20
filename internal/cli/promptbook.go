package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/pkg/alaws"
)

func newPromptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Manage prompt templates within a book",
	}
	cmd.AddCommand(
		newPromptCreateCmd(),
		newPromptListCmd(),
		newPromptShowCmd(),
		newPromptVarsCmd(),
		newPromptRenderCmd(),
		newPromptRemoveCmd(),
		newPromptMoveCmd(),
	)
	return cmd
}

func newPromptCreateCmd() *cobra.Command {
	var bookFlag, title, id, after, before string
	var position int
	cmd := &cobra.Command{
		Use:   "create <file>",
		Short: "Create a new prompt template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			p := alaws.Placement{After: after, Before: before, Position: position}
			if flagDryRun {
				cmd.Printf("would create %s/%s and insert into promptTemplates\n", book, file)
				return nil
			}
			if err := alaws.CreatePrompt(book, file, title, id, p); err != nil {
				return err
			}
			cmd.Printf("created prompt %s (%s)\n", id, file)
			return nil
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().StringVar(&title, "title", "", "prompt title (required)")
	cmd.Flags().StringVar(&id, "id", "", "stable prompt ID (required)")
	cmd.Flags().StringVar(&before, "before", "", "insert before this prompt file")
	cmd.Flags().StringVar(&after, "after", "", "insert after this prompt file")
	cmd.Flags().IntVar(&position, "position", 0, "insert at this 1-based position")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("id")
	return cmd
}

func newPromptListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [book]",
		Short: "List prompt templates in a book",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			prompts := b.Prompts()
			return printResult(cmd, prompts, func() {
				if len(prompts) == 0 {
					cmd.Println("No prompt templates.")
					return
				}
				for _, p := range prompts {
					cmd.Printf("%-40s %s\n", p.ID, p.Title)
				}
			})
		},
	}
	return cmd
}

func newPromptShowCmd() *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "show <book> <id>",
		Short: "Show a prompt template's details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, id := args[0], args[1]
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			p, err := b.Prompt(id)
			if err != nil {
				return err
			}
			return printResult(cmd, p.PromptTemplate, func() {
				cmd.Printf("ID:    %s\n", p.ID)
				cmd.Printf("Title: %s\n", p.Title)
				cmd.Printf("Vars:  %s\n", strings.Join(p.Vars, ", "))
				cmd.Println()
				if p.Commentary != "" {
					cmd.Println("Commentary:")
					cmd.Println(p.Commentary)
					cmd.Println()
				}
				cmd.Println("Template:")
				if raw {
					cmd.Println(p.Template)
				} else {
					for _, seg := range p.Segments {
						switch seg.Kind {
						case 0: // SegmentText
							cmd.Print(seg.Text)
						default:
							cmd.Printf("{{ref:%s}} → %s\n", seg.RefToken, seg.RefLabel)
						}
					}
				}
			})
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "show the fully expanded template")
	return cmd
}

func newPromptVarsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vars <book> <id>",
		Short: "List the variables a prompt template requires",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, id := args[0], args[1]
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			p, err := b.Prompt(id)
			if err != nil {
				return err
			}
			return printResult(cmd, p.Vars, func() {
				if len(p.Vars) == 0 {
					cmd.Println("No variables required.")
					return
				}
				for _, v := range p.Vars {
					cmd.Println(v)
				}
			})
		},
	}
	return cmd
}

func newPromptRenderCmd() *cobra.Command {
	var onMissing string
	var vars []string
	cmd := &cobra.Command{
		Use:   "render <book> <id>",
		Short: "Render a prompt template with variable substitution",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, id := args[0], args[1]
			b, err := alaws.Load(book)
			if err != nil {
				return err
			}
			p, err := b.Prompt(id)
			if err != nil {
				return err
			}

			varMap := map[string]string{}
			for _, kv := range vars {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("--var must be in key=value form, got %q", kv)
				}
				varMap[k] = v
			}

			policy := alaws.MissingError
			switch onMissing {
			case "keep":
				policy = alaws.MissingKeepPlaceholder
			case "empty":
				policy = alaws.MissingEmpty
			}

			text, err := p.Render(alaws.PromptRenderOptions{Vars: varMap, OnMissing: policy})
			if err != nil {
				return err
			}
			return printResult(cmd, map[string]string{"text": text}, func() {
				cmd.Println(text)
			})
		},
	}
	cmd.Flags().StringSliceVar(&vars, "var", nil, "variable in key=value form (repeatable)")
	cmd.Flags().StringVar(&onMissing, "on-missing", "error", "missing variable policy: error, keep, empty")
	return cmd
}

func newPromptRemoveCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "remove <book> <id>",
		Short: "Remove a prompt template from the book",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, id := args[0], args[1]
			if flagDryRun {
				cmd.Printf("would remove prompt %s from %s\n", id, book)
				return nil
			}
			// Find the file path for this prompt ID
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			var filePath string
			for _, p := range b.Prompts() {
				if p.ID == id {
					filePath = p.Source.Path
					break
				}
			}
			if filePath == "" {
				return fmt.Errorf("prompt %q not found", id)
			}
			// Make path relative to book
			if strings.HasPrefix(filePath, book+"/") {
				filePath = filePath[len(book)+1:]
			}
			if err := alaws.RemovePrompt(book, filePath); err != nil {
				return err
			}
			cmd.Printf("removed prompt %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}

func newPromptMoveCmd() *cobra.Command {
	var bookFlag, after, before string
	var position int
	cmd := &cobra.Command{
		Use:   "move <book> <file>",
		Short: "Move a prompt template to a new position",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, file := args[0], args[1]
			p := alaws.Placement{After: after, Before: before, Position: position}
			if flagDryRun {
				cmd.Printf("would move prompt %s in %s\n", file, book)
				return nil
			}
			if err := alaws.MovePrompt(book, file, p); err != nil {
				return err
			}
			cmd.Printf("moved prompt %s\n", file)
			return nil
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().StringVar(&before, "before", "", "move before this prompt file")
	cmd.Flags().StringVar(&after, "after", "", "move after this prompt file")
	cmd.Flags().IntVar(&position, "position", 0, "move to this 1-based position")
	return cmd
}
