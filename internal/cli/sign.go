package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/internal/provenance"
	"github.com/shrsv/AgentLaws/internal/signing"
	"github.com/shrsv/AgentLaws/pkg/alaws"
)

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
			canonical, err := json.Marshal(b.Lawbook())
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
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			manifest, err := provenance.BuildManifest(b.Lawbook())
			if err != nil {
				return err
			}
			canonical, err := json.Marshal(b.Lawbook())
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
