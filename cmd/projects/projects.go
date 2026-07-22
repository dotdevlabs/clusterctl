// Package projects provides the "projects" subcommand tree for clusterctl.
package projects

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"
)

// Project is the API response shape for a project resource.
type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type createProjectRequest struct {
	Name string `json:"name"`
}

var projectCols = []output.Column{
	{Header: "ID"},
	{Header: "NAME"},
	{Header: "CREATED"},
}

func projectRow(p Project) []string {
	return []string{p.ID, p.Name, p.CreatedAt}
}

// NewCommand returns the "projects" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Manage projects",
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
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			env, err := httpclient.GetEnvelope[[]Project](cmd.Context(), client, "/api/v1/projects")
			if err != nil {
				return err
			}
			var rows [][]string
			for _, p := range env.Data {
				rows = append(rows, projectRow(p))
			}
			return renderer.Render(projectCols, rows, env)
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a project by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/projects/" + url.PathEscape(args[0])
			env, err := httpclient.GetEnvelope[Project](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			return renderer.Render(projectCols, [][]string{projectRow(env.Data)}, env)
		},
	}
}

func newCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := createProjectRequest{Name: name}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			env, err := httpclient.PostEnvelope[Project](cmd.Context(), client, "/api/v1/projects", body)
			if err != nil {
				return err
			}
			return renderer.Render(projectCols, [][]string{projectRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if len(body) == 0 {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			path := "/api/v1/projects/" + url.PathEscape(args[0])
			env, err := patchEnvelope[Project](cmd.Context(), client, path, body)
			if err != nil {
				return err
			}
			return renderer.Render(projectCols, [][]string{projectRow(env.Data)}, env)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := cmd.OutOrStdout().Write([]byte("DELETE /api/v1/projects/" + url.PathEscape(args[0]) + "\n"))
				return err
			}
			return client.Delete(cmd.Context(), "/api/v1/projects/"+url.PathEscape(args[0]))
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
