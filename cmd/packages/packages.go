// Package packages provides the "packages" subcommand tree for clusterctl.
package packages

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

const packageResourceType = "packages"

// Package is the API response shape for a package resource.
type Package struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	SourceType   string `json:"source_type,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	SourceBranch string `json:"source_branch,omitempty"`
	SourcePath   string `json:"source_path,omitempty"`
	SourceChart  string `json:"source_chart,omitempty"`
	SourceTagPat string `json:"source_tag_pattern,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type packageAttrs struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	SourceType   string `json:"source_type,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	SourceBranch string `json:"source_branch,omitempty"`
	SourcePath   string `json:"source_path,omitempty"`
	SourceChart  string `json:"source_chart,omitempty"`
	SourceTagPat string `json:"source_tag_pattern,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

func packageFromResource(r httpclient.Resource[packageAttrs]) Package {
	a := r.Attributes
	return Package{
		ID:           r.ID,
		Name:         a.Name,
		Description:  a.Description,
		SourceType:   a.SourceType,
		SourceURL:    a.SourceURL,
		SourceBranch: a.SourceBranch,
		SourcePath:   a.SourcePath,
		SourceChart:  a.SourceChart,
		SourceTagPat: a.SourceTagPat,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// packageRequestAttrs matches PackageRequest.data.attributes in the spec.
type packageRequestAttrs struct {
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	SourceType   string `json:"source_type,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	SourceBranch string `json:"source_branch,omitempty"`
	SourcePath   string `json:"source_path,omitempty"`
	SourceChart  string `json:"source_chart,omitempty"`
	SourceTagPat string `json:"source_tag_pattern,omitempty"`
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
			resources, err := jsonapi.GetAllPages[packageAttrs](cmd.Context(), client, "/api/v1/packages")
			if err != nil {
				return err
			}
			var rows [][]string
			var items []Package
			for _, r := range resources {
				p := packageFromResource(r)
				items = append(items, p)
				rows = append(rows, packageRow(p))
			}
			return renderer.Render(packageCols, rows, httpclient.Envelope[[]Package]{Data: items})
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
			res, err := jsonapi.GetSingle[packageAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			p := packageFromResource(res.Resource)
			return renderer.Render(packageCols, [][]string{packageRow(p)}, httpclient.Envelope[Package]{Data: p})
		},
	}
}

func newCreateCmd() *cobra.Command {
	var name, description, sourceType, sourceURL, sourceBranch, sourcePath, sourceChart, sourceTagPat string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new package",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := jsonapi.Wrap(packageResourceType, packageRequestAttrs{
				Name:         name,
				Description:  description,
				SourceType:   sourceType,
				SourceURL:    sourceURL,
				SourceBranch: sourceBranch,
				SourcePath:   sourcePath,
				SourceChart:  sourceChart,
				SourceTagPat: sourceTagPat,
			})
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			res, err := httpclient.PostJSONAPISingle[packageAttrs](cmd.Context(), client, "/api/v1/packages", body)
			if err != nil {
				return err
			}
			p := packageFromResource(res)
			return renderer.Render(packageCols, [][]string{packageRow(p)}, httpclient.Envelope[Package]{Data: p})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Package name")
	cmd.Flags().StringVar(&description, "description", "", "Package description")
	cmd.Flags().StringVar(&sourceType, "source-type", "", "Source type (helm|git)")
	cmd.Flags().StringVar(&sourceURL, "source-url", "", "Source URL")
	cmd.Flags().StringVar(&sourceBranch, "source-branch", "", "Source branch")
	cmd.Flags().StringVar(&sourcePath, "source-path", "", "Source path")
	cmd.Flags().StringVar(&sourceChart, "source-chart", "", "Source chart name")
	cmd.Flags().StringVar(&sourceTagPat, "source-tag-pattern", "", "Source tag pattern")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var name, description, sourceType, sourceURL, sourceBranch, sourcePath, sourceChart, sourceTagPat string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := packageRequestAttrs{}
			anyChanged := false
			if cmd.Flags().Changed("name") {
				attrs.Name = name
				anyChanged = true
			}
			if cmd.Flags().Changed("description") {
				attrs.Description = description
				anyChanged = true
			}
			if cmd.Flags().Changed("source-type") {
				attrs.SourceType = sourceType
				anyChanged = true
			}
			if cmd.Flags().Changed("source-url") {
				attrs.SourceURL = sourceURL
				anyChanged = true
			}
			if cmd.Flags().Changed("source-branch") {
				attrs.SourceBranch = sourceBranch
				anyChanged = true
			}
			if cmd.Flags().Changed("source-path") {
				attrs.SourcePath = sourcePath
				anyChanged = true
			}
			if cmd.Flags().Changed("source-chart") {
				attrs.SourceChart = sourceChart
				anyChanged = true
			}
			if cmd.Flags().Changed("source-tag-pattern") {
				attrs.SourceTagPat = sourceTagPat
				anyChanged = true
			}
			if !anyChanged {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			body := jsonapi.Wrap(packageResourceType, attrs)
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			initialPath := "/api/v1/packages/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[packageAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			selfPath := jsonapi.SelfPath(fetched.SelfLink, initialPath)
			res, err := jsonapi.PatchSingle[packageAttrs](cmd.Context(), client, selfPath, body)
			if err != nil {
				return err
			}
			p := packageFromResource(res)
			return renderer.Render(packageCols, [][]string{packageRow(p)}, httpclient.Envelope[Package]{Data: p})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Package name")
	cmd.Flags().StringVar(&description, "description", "", "Package description")
	cmd.Flags().StringVar(&sourceType, "source-type", "", "Source type")
	cmd.Flags().StringVar(&sourceURL, "source-url", "", "Source URL")
	cmd.Flags().StringVar(&sourceBranch, "source-branch", "", "Source branch")
	cmd.Flags().StringVar(&sourcePath, "source-path", "", "Source path")
	cmd.Flags().StringVar(&sourceChart, "source-chart", "", "Source chart name")
	cmd.Flags().StringVar(&sourceTagPat, "source-tag-pattern", "", "Source tag pattern")
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
			initialPath := "/api/v1/packages/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[packageAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			return client.Delete(cmd.Context(), jsonapi.SelfPath(fetched.SelfLink, initialPath))
		},
	}
}
