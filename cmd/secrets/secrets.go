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

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

const secretResourceType = "project_secrets"

// Secret is the API response shape for a secret resource.
type Secret struct {
	ID                   string `json:"id"`
	KubernetesSecretName string `json:"kubernetes_secret_name,omitempty"`
	Key                  string `json:"key,omitempty"`
	CreatedAt            string `json:"created_at,omitempty"`
}

type secretAttrs struct {
	KubernetesSecretName string `json:"kubernetes_secret_name,omitempty"`
	Key                  string `json:"key,omitempty"`
	CreatedAt            string `json:"created_at,omitempty"`
}

func secretFromResource(r httpclient.Resource[secretAttrs]) Secret {
	a := r.Attributes
	return Secret{
		ID:                   r.ID,
		KubernetesSecretName: a.KubernetesSecretName,
		Key:                  a.Key,
		CreatedAt:            a.CreatedAt,
	}
}

// createSecretAttrs matches the SecretRequest.data.attributes in the spec.
type createSecretAttrs struct {
	KubernetesSecretName string `json:"kubernetes_secret_name"`
	Key                  string `json:"key"`
	Value                string `json:"value"`
}

// secretMaterializationAttrs matches the SecretMaterializationAttributes in the spec.
type secretMaterializationAttrs struct {
	AppliedCount int    `json:"applied_count"`
	Message      string `json:"message"`
}

var secretCols = []output.Column{
	{Header: "ID"},
	{Header: "K8S-SECRET"},
	{Header: "KEY"},
	{Header: "CREATED"},
}

func secretRow(s Secret) []string {
	return []string{s.ID, s.KubernetesSecretName, s.Key, s.CreatedAt}
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
			col, err := httpclient.GetJSONAPICollection[secretAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			var rows [][]string
			var items []Secret
			for _, r := range col.Data {
				s := secretFromResource(r)
				items = append(items, s)
				rows = append(rows, secretRow(s))
			}
			return renderer.Render(secretCols, rows, httpclient.Envelope[[]Secret]{Data: items})
		},
	}
}

func newCreateCmd(projectID *string) *cobra.Command {
	var secretName, key, value string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a secret entry in a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if *projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := jsonapi.Wrap(secretResourceType, createSecretAttrs{
				KubernetesSecretName: secretName,
				Key:                  key,
				Value:                value,
			})
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			path := "/api/v1/projects/" + url.PathEscape(*projectID) + "/secrets"
			res, err := httpclient.PostJSONAPISingle[secretAttrs](cmd.Context(), client, path, body)
			if err != nil {
				return err
			}
			s := secretFromResource(res)
			return renderer.Render(secretCols, [][]string{secretRow(s)}, httpclient.Envelope[Secret]{Data: s})
		},
	}
	cmd.Flags().StringVar(&secretName, "secret-name", "", "Target Kubernetes Secret name (kubernetes_secret_name)")
	cmd.Flags().StringVar(&key, "key", "", "Key within the Kubernetes Secret data map")
	cmd.Flags().StringVar(&value, "value", "", "Secret value")
	if err := cmd.MarkFlagRequired("secret-name"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("key"); err != nil {
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
			res, err := httpclient.PostJSONAPISingle[secretMaterializationAttrs](cmd.Context(), client, path, nil)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Attributes)
		},
	}
}
