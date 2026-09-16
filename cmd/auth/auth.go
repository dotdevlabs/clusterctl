package auth

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

type authContextAttrs struct {
	Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"organization"`
	Owner struct {
		ID           string `json:"id"`
		EmailAddress string `json:"email_address"`
	} `json:"owner"`
	Token struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ExpiresAt string `json:"expires_at,omitempty"`
	} `json:"token"`
}

// NewCommand returns the "auth" cobra.Command with all subcommands attached.
// Used by the conformance test's isolated root builder.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with ClusterControl",
	}
	cmd.AddCommand(NewWhoamiCmd())
	return cmd
}

// NewWhoamiCmd returns the "whoami" subcommand so it can be attached to an
// existing auth command (e.g. ctlkit's built-in auth command in the real binary).
func NewWhoamiCmd() *cobra.Command {
	return newWhoamiCmd()
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authentication context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			res, err := jsonapi.GetSingle[authContextAttrs](cmd.Context(), client, "/api/v1/auth")
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Resource.Attributes)
		},
	}
}
