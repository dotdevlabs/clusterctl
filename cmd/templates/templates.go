package templates

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

type templateAttrs struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ResourceNames []string `json:"resource_names,omitempty"`
}

type Template struct {
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ResourceNames []string `json:"resource_names,omitempty"`
}

var templateCols = []output.Column{
	{Header: "ID"},
	{Header: "SLUG"},
	{Header: "NAME"},
}

func templateFromResource(r httpclient.Resource[templateAttrs]) Template {
	a := r.Attributes
	return Template{
		ID:            r.ID,
		Slug:          a.Slug,
		Name:          a.Name,
		Description:   a.Description,
		ResourceNames: a.ResourceNames,
	}
}

func templateRow(t Template) []string {
	return []string{t.ID, t.Slug, t.Name}
}

// NewCommand returns the "templates" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Manage templates",
	}
	cmd.AddCommand(newListCmd(), newGetCmd())
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			resources, err := jsonapi.GetAllPages[templateAttrs](cmd.Context(), client, "/api/v1/templates")
			if err != nil {
				return err
			}
			var rows [][]string
			var items []Template
			for _, r := range resources {
				tmpl := templateFromResource(r)
				items = append(items, tmpl)
				rows = append(rows, templateRow(tmpl))
			}
			return renderer.Render(templateCols, rows, httpclient.Envelope[[]Template]{Data: items})
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a template by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/templates/" + url.PathEscape(args[0])
			res, err := jsonapi.GetSingle[templateAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			tmpl := templateFromResource(res.Resource)
			return renderer.Render(templateCols, [][]string{templateRow(tmpl)}, httpclient.Envelope[Template]{Data: tmpl})
		},
	}
}
