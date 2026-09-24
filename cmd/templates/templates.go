package templates

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

// TemplateInput is one element of the inputs array (read + write).
type TemplateInput struct {
	Key         string          `json:"key"`
	Type        string          `json:"type"`
	Default     json.RawMessage `json:"default"`
	Required    bool            `json:"required"`
	Description *string         `json:"description"`
}

type templateAttrs struct {
	Slug          string            `json:"slug"`
	Name          string            `json:"name"`
	Description   *string           `json:"description"`
	ResourceNames []string          `json:"resource_names,omitempty"`
	ManifestFiles map[string]string `json:"manifest_files,omitempty"`
	Inputs        []TemplateInput   `json:"inputs,omitempty"`
}

// Template is the stable CLI output type for a template resource.
type Template struct {
	ID            string            `json:"id"`
	Slug          string            `json:"slug"`
	Name          string            `json:"name"`
	Description   *string           `json:"description,omitempty"`
	ResourceNames []string          `json:"resource_names,omitempty"`
	ManifestFiles map[string]string `json:"manifest_files,omitempty"`
	Inputs        []TemplateInput   `json:"inputs,omitempty"`
}

// templateCreateAttrs maps POST /templates request body attributes.
// slug and name are required; others are optional per the published schema.
// resource_names and derived are intentionally absent (not in published create schema).
type templateCreateAttrs struct {
	Slug          string          `json:"slug"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Inputs        json.RawMessage `json:"inputs,omitempty"`
	ManifestFiles json.RawMessage `json:"manifest_files,omitempty"`
}

// templateUpdateAttrs maps PATCH /templates/{id} request body attributes.
// Only description, inputs, manifest_files are mutable per the published schema.
// json.RawMessage with omitempty: nil → field omitted; non-nil → sent (including "null"/"[]"/"{}").
type templateUpdateAttrs struct {
	Description   json.RawMessage `json:"description,omitempty"`
	Inputs        json.RawMessage `json:"inputs,omitempty"`
	ManifestFiles json.RawMessage `json:"manifest_files,omitempty"`
}

// templateUpdateMeta is data.meta of the PATCH /templates/{id} 200 response.
type templateUpdateMeta struct {
	RerendereredDeployments []string `json:"rerendered_deployments"`
	RerenderErrors          []string `json:"rerender_errors"`
}

// TemplateUpdateResult is the stable JSON output for the update command.
type TemplateUpdateResult struct {
	Data                    Template `json:"data"`
	RerendereredDeployments []string `json:"rerendered_deployments"`
	RerenderErrors          []string `json:"rerender_errors"`
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
		ManifestFiles: a.ManifestFiles,
		Inputs:        a.Inputs,
	}
}

func templateRow(t Template) []string {
	return []string{t.ID, t.Slug, t.Name}
}

// readJSONArg returns raw JSON bytes for a string argument.
// If s starts with '@', the remainder is a file path whose content is read.
// Otherwise s is validated as JSON and returned as-is.
func readJSONArg(s string) (json.RawMessage, error) {
	var raw []byte
	if strings.HasPrefix(s, "@") {
		b, err := os.ReadFile(s[1:]) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("reading file %q: %w", s[1:], err)
		}
		raw = b
	} else {
		raw = []byte(s)
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid JSON")
	}
	return json.RawMessage(raw), nil
}

// NewCommand returns the "templates" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Manage templates",
	}
	cmd.AddCommand(newListCmd(), newGetCmd(), newCreateCmd(), newUpdateCmd())
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

func newCreateCmd() *cobra.Command {
	var slug, name string
	var description string
	var inputsJSON, manifestFilesJSON string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a deployment template",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := templateCreateAttrs{
				Slug: slug,
				Name: name,
			}
			if cmd.Flags().Changed("description") {
				attrs.Description = &description
			}
			if cmd.Flags().Changed("inputs") {
				raw, err := readJSONArg(inputsJSON)
				if err != nil {
					return fmt.Errorf("--inputs: %w", err)
				}
				attrs.Inputs = raw
			}
			if cmd.Flags().Changed("manifest-files") {
				raw, err := readJSONArg(manifestFilesJSON)
				if err != nil {
					return fmt.Errorf("--manifest-files: %w", err)
				}
				attrs.ManifestFiles = raw
			}

			body := jsonapi.Wrap("templates", attrs)
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			res, err := httpclient.PostJSONAPISingle[templateAttrs](cmd.Context(), client, "/api/v1/templates", body)
			if err != nil {
				return err
			}
			tmpl := templateFromResource(res)
			return renderer.Render(templateCols, [][]string{templateRow(tmpl)}, httpclient.Envelope[Template]{Data: tmpl})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "Template slug (required)")
	cmd.Flags().StringVar(&name, "name", "", "Template name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Template description")
	cmd.Flags().StringVar(&inputsJSON, "inputs", "", "Inputs as JSON array or @file (e.g. '[{\"key\":\"image\",\"type\":\"string\",\"required\":true}]')")
	cmd.Flags().StringVar(&manifestFilesJSON, "manifest-files", "", "Manifest files as JSON object mapping filename to content, or @file")
	if err := cmd.MarkFlagRequired("slug"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var description string
	var clearDescription bool
	var inputsJSON, manifestFilesJSON string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a deployment template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := templateUpdateAttrs{}
			anyChanged := false

			if clearDescription {
				attrs.Description = json.RawMessage("null")
				anyChanged = true
			} else if cmd.Flags().Changed("description") {
				b, err := json.Marshal(description)
				if err != nil {
					return fmt.Errorf("encoding description: %w", err)
				}
				attrs.Description = b
				anyChanged = true
			}
			if cmd.Flags().Changed("inputs") {
				raw, err := readJSONArg(inputsJSON)
				if err != nil {
					return fmt.Errorf("--inputs: %w", err)
				}
				attrs.Inputs = raw
				anyChanged = true
			}
			if cmd.Flags().Changed("manifest-files") {
				raw, err := readJSONArg(manifestFilesJSON)
				if err != nil {
					return fmt.Errorf("--manifest-files: %w", err)
				}
				attrs.ManifestFiles = raw
				anyChanged = true
			}

			if !anyChanged {
				return fmt.Errorf("no update flags provided; specify at least one of --description, --clear-description, --inputs, --manifest-files")
			}

			path := "/api/v1/templates/" + url.PathEscape(args[0])
			body := jsonapi.Wrap("templates", attrs)
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			res, err := jsonapi.PatchSingleWithMeta[templateAttrs, templateUpdateMeta](cmd.Context(), client, path, body)
			if err != nil {
				return err
			}
			tmpl := templateFromResource(res.Resource)
			result := TemplateUpdateResult{
				Data:                    tmpl,
				RerendereredDeployments: res.Meta.RerendereredDeployments,
				RerenderErrors:          res.Meta.RerenderErrors,
			}
			// In JSON mode renderer outputs result directly; in table mode it outputs the table rows.
			// Rerender errors are part of the JSON struct; in table mode they are also printed to stderr.
			if err := renderer.Render(templateCols, [][]string{templateRow(tmpl)}, result); err != nil {
				return err
			}
			if !gf.JSON {
				for _, e := range res.Meta.RerenderErrors {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: rerender error: %s\n", e)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "New template description")
	cmd.Flags().BoolVar(&clearDescription, "clear-description", false, "Set description to null")
	cmd.Flags().StringVar(&inputsJSON, "inputs", "", "Inputs as JSON array or @file")
	cmd.Flags().StringVar(&manifestFilesJSON, "manifest-files", "", "Manifest files as JSON object or @file")
	return cmd
}
