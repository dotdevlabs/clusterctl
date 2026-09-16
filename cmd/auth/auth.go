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
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with ClusterControl",
	}
	cmd.AddCommand(newWhoamiCmd())
	return cmd
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
