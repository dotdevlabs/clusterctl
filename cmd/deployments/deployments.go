// Package deployments provides the "deployments" subcommand tree for clusterctl.
package deployments

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"
)

// Deployment is the API response shape for a deployment resource.
type Deployment struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id,omitempty"`
	ClusterID      string `json:"cluster_id,omitempty"`
	PackageID      string `json:"package_id,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	ValuesOverride string `json:"values_override,omitempty"`
	Status         string `json:"status,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

type createDeploymentRequest struct {
	ProjectID      string `json:"project_id,omitempty"`
	ClusterID      string `json:"cluster_id,omitempty"`
	PackageID      string `json:"package_id,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	ValuesOverride string `json:"values_override,omitempty"`
}

var deploymentCols = []output.Column{
	{Header: "ID"},
	{Header: "PROJECT"},
	{Header: "CLUSTER"},
	{Header: "PACKAGE"},
	{Header: "STATUS"},
}

func deploymentRow(d Deployment) []string {
	return []string{d.ID, d.ProjectID, d.ClusterID, d.PackageID, d.Status}
}

// NewCommand returns the "deployments" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployments",
		Short: "Manage deployments",
	}
	cmd.AddCommand(
		newListCmd(),
		newGetCmd(),
		newCreateCmd(),
		newUpdateCmd(),
		newDeleteCmd(),
	)
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all deployments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			env, err := httpclient.GetEnvelope[[]Deployment](cmd.Context(), client, "/api/v1/deployments")
			if err != nil {
				return err
			}
			var rows [][]string
			for _, d := range env.Data {
				rows = append(rows, deploymentRow(d))
			}
			return renderer.Render(deploymentCols, rows, env)
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a deployment by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/deployments/" + url.PathEscape(args[0])
			env, err := httpclient.GetEnvelope[Deployment](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			return renderer.Render(deploymentCols, [][]string{deploymentRow(env.Data)}, env)
		},
	}
}

func newCreateCmd() *cobra.Command {
	var projectID, clusterID, packageID, packageVersion, valuesOverride string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new deployment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := createDeploymentRequest{
				ProjectID:      projectID,
				ClusterID:      clusterID,
				PackageID:      packageID,
				PackageVersion: packageVersion,
				ValuesOverride: valuesOverride,
			}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			env, err := httpclient.PostEnvelope[Deployment](cmd.Context(), client, "/api/v1/deployments", body)
			if err != nil {
				return err
			}
			return renderer.Render(deploymentCols, [][]string{deploymentRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "Cluster ID")
	cmd.Flags().StringVar(&packageID, "package-id", "", "Package ID")
	cmd.Flags().StringVar(&packageVersion, "package-version", "", "Package version")
	cmd.Flags().StringVar(&valuesOverride, "values-override", "", "Values override (YAML/JSON string)")
	if err := cmd.MarkFlagRequired("project-id"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("cluster-id"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("package-id"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var projectID, clusterID, packageID, packageVersion, valuesOverride string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := map[string]any{}
			if cmd.Flags().Changed("project-id") {
				body["project_id"] = projectID
			}
			if cmd.Flags().Changed("cluster-id") {
				body["cluster_id"] = clusterID
			}
			if cmd.Flags().Changed("package-id") {
				body["package_id"] = packageID
			}
			if cmd.Flags().Changed("package-version") {
				body["package_version"] = packageVersion
			}
			if cmd.Flags().Changed("values-override") {
				body["values_override"] = valuesOverride
			}
			if len(body) == 0 {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			path := "/api/v1/deployments/" + url.PathEscape(args[0])
			env, err := patchEnvelope[Deployment](cmd.Context(), client, path, body)
			if err != nil {
				return err
			}
			return renderer.Render(deploymentCols, [][]string{deploymentRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "Cluster ID")
	cmd.Flags().StringVar(&packageID, "package-id", "", "Package ID")
	cmd.Flags().StringVar(&packageVersion, "package-version", "", "Package version")
	cmd.Flags().StringVar(&valuesOverride, "values-override", "", "Values override (YAML/JSON string)")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := cmd.OutOrStdout().Write([]byte("DELETE /api/v1/deployments/" + url.PathEscape(args[0]) + "\n"))
				return err
			}
			return client.Delete(cmd.Context(), "/api/v1/deployments/"+url.PathEscape(args[0]))
		},
	}
}

func patchEnvelope[T any](ctx context.Context, client *httpclient.Client, path string, body any) (httpclient.Envelope[T], error) {
	var env httpclient.Envelope[T]
	if err := client.Patch(ctx, path, body, &env); err != nil {
		return httpclient.Envelope[T]{}, err
	}
	return env, nil
}
