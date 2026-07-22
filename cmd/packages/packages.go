// Package packages provides the "packages" subcommand tree for clusterctl.
package packages

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"
)

// Package is the API response shape for a package resource.
type Package struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SourceType   string `json:"source_type,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	SourceBranch string `json:"source_branch,omitempty"`
	SourcePath   string `json:"source_path,omitempty"`
	SourceChart  string `json:"source_chart,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type createPackageRequest struct {
	Name         string `json:"name"`
	SourceType   string `json:"source_type,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	SourceBranch string `json:"source_branch,omitempty"`
	SourcePath   string `json:"source_path,omitempty"`
	SourceChart  string `json:"source_chart,omitempty"`
}

var packageCols = []output.Column{
	{Header: "ID"},
	{Header: "NAME"},
	{Header: "SOURCE_TYPE"},
	{Header: "SOURCE_URL"},
}

func packageRow(p Package) []string {
	return []string{p.ID, p.Name, p.SourceType, p.SourceURL}
}

// NewCommand returns the "packages" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packages",
		Short: "Manage packages",
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
		Short: "List all packages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			env, err := httpclient.GetEnvelope[[]Package](cmd.Context(), client, "/api/v1/packages")
			if err != nil {
				return err
			}
			var rows [][]string
			for _, p := range env.Data {
				rows = append(rows, packageRow(p))
			}
			return renderer.Render(packageCols, rows, env)
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a package by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/packages/" + url.PathEscape(args[0])
			env, err := httpclient.GetEnvelope[Package](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			return renderer.Render(packageCols, [][]string{packageRow(env.Data)}, env)
		},
	}
}

func newCreateCmd() *cobra.Command {
	var name, sourceType, sourceURL, sourceBranch, sourcePath, sourceChart string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new package",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := createPackageRequest{
				Name:         name,
				SourceType:   sourceType,
				SourceURL:    sourceURL,
				SourceBranch: sourceBranch,
				SourcePath:   sourcePath,
				SourceChart:  sourceChart,
			}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			env, err := httpclient.PostEnvelope[Package](cmd.Context(), client, "/api/v1/packages", body)
			if err != nil {
				return err
			}
			return renderer.Render(packageCols, [][]string{packageRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Package name")
	cmd.Flags().StringVar(&sourceType, "source-type", "", "Source type (helm|git)")
	cmd.Flags().StringVar(&sourceURL, "source-url", "", "Source URL")
	cmd.Flags().StringVar(&sourceBranch, "source-branch", "", "Source branch")
	cmd.Flags().StringVar(&sourcePath, "source-path", "", "Source path")
	cmd.Flags().StringVar(&sourceChart, "source-chart", "", "Source chart name")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var name, sourceType, sourceURL, sourceBranch, sourcePath, sourceChart string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("source-type") {
				body["source_type"] = sourceType
			}
			if cmd.Flags().Changed("source-url") {
				body["source_url"] = sourceURL
			}
			if cmd.Flags().Changed("source-branch") {
				body["source_branch"] = sourceBranch
			}
			if cmd.Flags().Changed("source-path") {
				body["source_path"] = sourcePath
			}
			if cmd.Flags().Changed("source-chart") {
				body["source_chart"] = sourceChart
			}
			if len(body) == 0 {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			path := "/api/v1/packages/" + url.PathEscape(args[0])
			env, err := patchEnvelope[Package](cmd.Context(), client, path, body)
			if err != nil {
				return err
			}
			return renderer.Render(packageCols, [][]string{packageRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Package name")
	cmd.Flags().StringVar(&sourceType, "source-type", "", "Source type")
	cmd.Flags().StringVar(&sourceURL, "source-url", "", "Source URL")
	cmd.Flags().StringVar(&sourceBranch, "source-branch", "", "Source branch")
	cmd.Flags().StringVar(&sourcePath, "source-path", "", "Source path")
	cmd.Flags().StringVar(&sourceChart, "source-chart", "", "Source chart name")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := cmd.OutOrStdout().Write([]byte("DELETE /api/v1/packages/" + url.PathEscape(args[0]) + "\n"))
				return err
			}
			return client.Delete(cmd.Context(), "/api/v1/packages/"+url.PathEscape(args[0]))
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
