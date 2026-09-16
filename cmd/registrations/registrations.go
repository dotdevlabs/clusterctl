package registrations

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
)

type registrationRequest struct {
	OwnerEmail string `json:"owner_email"`
	Label      string `json:"label"`
}

type registrationAttrs struct {
	Token          string `json:"token"`
	OrganizationID string `json:"organization_id"`
	OwnerID        string `json:"owner_id"`
}

// NewCommand returns the "registrations" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registrations",
		Short: "Manage registrations",
	}
	cmd.AddCommand(newCreateCmd())
	return cmd
}

func newCreateCmd() *cobra.Command {
	var ownerEmail, label string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new registration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			body := registrationRequest{
				OwnerEmail: ownerEmail,
				Label:      label,
			}
			res, err := httpclient.PostJSONAPISingle[registrationAttrs](cmd.Context(), client, "/api/v1/registrations", body)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Attributes)
		},
	}
	cmd.Flags().StringVar(&ownerEmail, "owner-email", "", "Owner email address")
	cmd.Flags().StringVar(&label, "label", "", "Registration label")
	if err := cmd.MarkFlagRequired("owner-email"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("label"); err != nil {
		panic(err)
	}
	return cmd
}
