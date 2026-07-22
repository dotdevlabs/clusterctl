// Package clusters provides the "clusters" subcommand tree for clusterctl.
package clusters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"
)

// Cluster is the API response shape for a cluster resource.
type Cluster struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ClusterType     string `json:"cluster_type"`
	ParentClusterID string `json:"parent_cluster_id,omitempty"`
	Status          string `json:"status,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

type createClusterRequest struct {
	Name            string `json:"name"`
	ClusterType     string `json:"cluster_type"`
	ParentClusterID string `json:"parent_cluster_id,omitempty"`
}

var clusterCols = []output.Column{
	{Header: "ID"},
	{Header: "NAME"},
	{Header: "TYPE"},
	{Header: "STATUS"},
	{Header: "CREATED"},
}

func clusterRow(c Cluster) []string {
	return []string{c.ID, c.Name, c.ClusterType, c.Status, c.CreatedAt}
}

// NewCommand returns the "clusters" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Manage clusters",
	}
	cmd.AddCommand(
		newListCmd(),
		newGetCmd(),
		newCreateCmd(),
		newUpdateCmd(),
		newDeleteCmd(),
		newHealthCheckCmd(),
		newFluxBootstrapCmd(),
	)
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all clusters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			env, err := httpclient.GetEnvelope[[]Cluster](cmd.Context(), client, "/api/v1/clusters")
			if err != nil {
				return err
			}
			var rows [][]string
			for _, c := range env.Data {
				rows = append(rows, clusterRow(c))
			}
			return renderer.Render(clusterCols, rows, env)
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a cluster by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/clusters/" + url.PathEscape(args[0])
			env, err := httpclient.GetEnvelope[Cluster](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			return renderer.Render(clusterCols, [][]string{clusterRow(env.Data)}, env)
		},
	}
}

func newCreateCmd() *cobra.Command {
	var name, clusterType, parentClusterID string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new cluster",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := createClusterRequest{
				Name:            name,
				ClusterType:     clusterType,
				ParentClusterID: parentClusterID,
			}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			env, err := httpclient.PostEnvelope[Cluster](cmd.Context(), client, "/api/v1/clusters", body)
			if err != nil {
				return err
			}
			return renderer.Render(clusterCols, [][]string{clusterRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&clusterType, "cluster-type", "", "Cluster type (virtual|imported)")
	cmd.Flags().StringVar(&parentClusterID, "parent-cluster-id", "", "Parent cluster ID (for virtual clusters)")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("cluster-type"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var name, clusterType, parentClusterID string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("cluster-type") {
				body["cluster_type"] = clusterType
			}
			if cmd.Flags().Changed("parent-cluster-id") {
				body["parent_cluster_id"] = parentClusterID
			}
			if len(body) == 0 {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			path := "/api/v1/clusters/" + url.PathEscape(args[0])
			env, err := patchEnvelope[Cluster](cmd.Context(), client, path, body)
			if err != nil {
				return err
			}
			return renderer.Render(clusterCols, [][]string{clusterRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&clusterType, "cluster-type", "", "Cluster type")
	cmd.Flags().StringVar(&parentClusterID, "parent-cluster-id", "", "Parent cluster ID")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "DELETE /api/v1/clusters/%s\n", url.PathEscape(args[0]))
				return err
			}
			return client.Delete(cmd.Context(), "/api/v1/clusters/"+url.PathEscape(args[0]))
		},
	}
}

func newHealthCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health-check <id>",
		Short: "Run a health check on a cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "POST /api/v1/clusters/%s/health_check\n", url.PathEscape(args[0]))
				return err
			}
			var resp map[string]any
			path := "/api/v1/clusters/" + url.PathEscape(args[0]) + "/health_check"
			if err := client.Post(cmd.Context(), path, nil, &resp); err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		},
	}
}

func newFluxBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "flux-bootstrap <id>",
		Short: "Run flux bootstrap on a cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "POST /api/v1/clusters/%s/flux_bootstrap\n", url.PathEscape(args[0]))
				return err
			}
			var resp map[string]any
			path := "/api/v1/clusters/" + url.PathEscape(args[0]) + "/flux_bootstrap"
			if err := client.Post(cmd.Context(), path, nil, &resp); err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
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
