package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/compiler"
	"github.com/athreyac4/agentlaws/internal/provenance"
	"github.com/athreyac4/agentlaws/internal/signing"
)

func newSignCmd() *cobra.Command {
	var key string
	cmd := &cobra.Command{
		Use:   "sign [book]",
		Short: "Sign the canonical representation of a compiled book",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book := flagRoot
			if len(args) == 1 {
				book = args[0]
			}
			result, err := compiler.Compile(book, compiler.Options{})
			if err != nil {
				return err
			}
			canonical, err := json.Marshal(result.Lawbook)
			if err != nil {
				return err
			}
			sig, err := signing.Sign(canonical, key)
			if err != nil {
				return err
			}
			cmd.Println(sig)
			return nil
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "signing key identity (defaults to the local Git identity)")
	return cmd
}

func newVerifyCmd() *cobra.Command {
	var manifestPath string
	cmd := &cobra.Command{
		Use:   "verify [book]",
		Short: "Verify a book's compiled state against its signed manifest",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book := flagRoot
			if len(args) == 1 {
				book = args[0]
			}
			result, err := compiler.Compile(book, compiler.Options{})
			if err != nil {
				return err
			}
			manifest, err := provenance.BuildManifest(result.Lawbook)
			if err != nil {
				return err
			}
			canonical, err := json.Marshal(result.Lawbook)
			if err != nil {
				return err
			}
			if err := signing.Verify(canonical, manifest.Signature); err != nil {
				return err
			}
			cmd.Println("verified")
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to an external manifest.json (defaults to the book's build output)")
	return cmd
}
