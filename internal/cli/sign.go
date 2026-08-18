package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/pkg/alaws"
)

// manifestPath is where `alaws sign` writes a book's manifest and where
// `alaws verify`/rendering look for one by default (docs/PLAN1.md §26).
func manifestPath(book string) string {
	return filepath.Join(book, ".alaws", "build", "manifest.json")
}

func newKeygenCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "keygen",
		Short: "Generate a new Ed25519 signing keypair",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := out
			if path == "" {
				var err error
				path, err = alaws.DefaultKeyPath()
				if err != nil {
					return err
				}
			}
			if flagDryRun {
				cmd.Printf("would write private key to %s (and %s.pub)\n", path, path)
				return nil
			}
			if err := alaws.GenerateKey(path); err != nil {
				return err
			}
			cmd.Printf("wrote %s and %s.pub\n", path, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "path for the private key (defaults to the §5 storage hierarchy)")
	return cmd
}

func newSignCmd() *cobra.Command {
	var key string
	cmd := &cobra.Command{
		Use:   "sign [book]",
		Short: "Sign the canonical representation of a compiled book",
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

			keyPath := key
			if keyPath == "" {
				keyPath, err = alaws.DefaultKeyPath()
				if err != nil {
					return err
				}
			}
			if _, statErr := os.Stat(keyPath); statErr != nil {
				return &UsageError{Msg: fmt.Sprintf("no signing key at %s - run `alaws keygen` first, or pass --key", keyPath)}
			}

			manifest, err := b.Sign(keyPath)
			if err != nil {
				return err
			}

			if !flagDryRun {
				out := manifestPath(book)
				if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
					return err
				}
				data, err := json.MarshalIndent(manifest, "", "  ")
				if err != nil {
					return err
				}
				if err := os.WriteFile(out, data, 0644); err != nil {
					return err
				}
			}

			return printResult(cmd, manifest, func() {
				cmd.Printf("signed %s\n  content hash: %s\n  signature:    %s\n",
					manifest.Lawbook, manifest.ContentHash, manifest.Signature)
			})
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "path to the private signing key (defaults to the §5 storage hierarchy)")
	return cmd
}

func newVerifyCmd() *cobra.Command {
	var manifestFlag string
	cmd := &cobra.Command{
		Use:   "verify [book]",
		Short: "Verify a book's compiled state against its signed manifest",
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

			path := manifestFlag
			if path == "" {
				path = manifestPath(book)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return &UsageError{Msg: fmt.Sprintf("no manifest at %s - run `alaws sign` first, or pass --manifest", path)}
			}
			var manifest alaws.Manifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return err
			}

			if err := alaws.Verify(manifest, b); err != nil {
				return err
			}
			cmd.Println("verified")
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestFlag, "manifest", "", "path to an external manifest.json (defaults to the book's build output)")
	return cmd
}
