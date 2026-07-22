// Package secrets provides the "secrets" subcommand tree for clusterctl.
package secrets

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"
)

// Secret is the API response shape for a secret resource.
type Secret struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"project_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type createSecretRequest struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

var secretCols = []output.Column{
	{Header: "ID"},
	{Header: "NAME"},
	{Header: "CREATED"},
}

func secretRow(s Secret) []string {
	return []string{s.ID, s.Name, s.CreatedAt}
}

// NewCommand returns the "secrets" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets",
	}
	cmd.PersistentFlags().StringVar(&projectID, "project-id", "", "Project ID")

	cmd.AddCommand(
		newListCmd(&projectID),
		newCreateCmd(&projectID),
		newDeleteCmd(&projectID),
		newMaterializeCmd(&projectID),
	)
	return cmd
}

func newListCmd(projectID *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List secrets in a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if *projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/projects/" + url.PathEscape(*projectID) + "/secrets"
			env, err := httpclient.GetEnvelope[[]Secret](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			var rows [][]string
			for _, s := range env.Data {
				rows = append(rows, secretRow(s))
			}
			return renderer.Render(secretCols, rows, env)
		},
	}
}

func newCreateCmd(projectID *string) *cobra.Command {
	var name, value string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a secret in a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if *projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := createSecretRequest{Name: name, Value: value}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			path := "/api/v1/projects/" + url.PathEscape(*projectID) + "/secrets"
			env, err := httpclient.PostEnvelope[Secret](cmd.Context(), client, path, body)
			if err != nil {
				return err
			}
			return renderer.Render(secretCols, [][]string{secretRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Secret name")
	cmd.Flags().StringVar(&value, "value", "", "Secret value")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	return cmd
}

func newDeleteCmd(projectID *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a secret from a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if *projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			path := "/api/v1/projects/" + url.PathEscape(*projectID) + "/secrets/" + url.PathEscape(args[0])
			if gf.DryRun {
				_, err := cmd.OutOrStdout().Write([]byte("DELETE " + path + "\n"))
				return err
			}
			return client.Delete(cmd.Context(), path)
		},
	}
}

func newMaterializeCmd(projectID *string) *cobra.Command {
	return &cobra.Command{
		Use:   "materialize",
		Short: "Materialize secrets to clusters in a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if *projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			path := "/api/v1/projects/" + url.PathEscape(*projectID) + "/secret_materialization"
			if gf.DryRun {
				_, err := cmd.OutOrStdout().Write([]byte("POST " + path + "\n"))
				return err
			}
			var resp map[string]any
			if err := client.Post(cmd.Context(), path, nil, &resp); err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		},
	}
}
