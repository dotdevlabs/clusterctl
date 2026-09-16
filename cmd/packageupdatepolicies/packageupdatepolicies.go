// Package packageupdatepolicies provides the "package-update-policies" subcommand tree for clusterctl.
package packageupdatepolicies

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

const policyResourceType = "package_update_policies"

// PackageUpdatePolicy is the API response shape for a package update policy resource.
type PackageUpdatePolicy struct {
	ID           string `json:"id"`
	DeploymentID string `json:"deployment_id,omitempty"`
	PackageID    string `json:"package_id,omitempty"`
	MaxAttempts  *int   `json:"max_attempts,omitempty"`
	IsBlocked    *bool  `json:"is_blocked,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type packageUpdatePolicyAttrs struct {
	DeploymentID string `json:"deployment_id,omitempty"`
	PackageID    string `json:"package_id,omitempty"`
	MaxAttempts  *int   `json:"max_attempts,omitempty"`
	IsBlocked    *bool  `json:"is_blocked,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

func policyFromResource(r httpclient.Resource[packageUpdatePolicyAttrs]) PackageUpdatePolicy {
	a := r.Attributes
	return PackageUpdatePolicy{
		ID:           r.ID,
		DeploymentID: a.DeploymentID,
		PackageID:    a.PackageID,
		MaxAttempts:  a.MaxAttempts,
		IsBlocked:    a.IsBlocked,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// packageUpdatePolicyWriteAttrs matches PackageUpdatePolicyRequest.data.attributes in the spec.
type packageUpdatePolicyWriteAttrs struct {
	DeploymentID string `json:"deployment_id,omitempty"`
	PackageID    string `json:"package_id,omitempty"`
	MaxAttempts  *int   `json:"max_attempts,omitempty"`
	IsBlocked    *bool  `json:"is_blocked,omitempty"`
}

var policyCols = []output.Column{
	{Header: "ID"},
	{Header: "DEPLOYMENT_ID"},
	{Header: "PACKAGE_ID"},
	{Header: "IS_BLOCKED"},
}

func policyRow(p PackageUpdatePolicy) []string {
	isBlocked := ""
	if p.IsBlocked != nil {
		if *p.IsBlocked {
			isBlocked = "true"
		} else {
			isBlocked = "false"
		}
	}
	return []string{p.ID, p.DeploymentID, p.PackageID, isBlocked}
}

// NewCommand returns the "package-update-policies" cobra.Command with all subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package-update-policies",
		Short: "Manage package update policies",
	}
	cmd.AddCommand(
		newListCmd(),
		newCreateCmd(),
		newGetCmd(),
		newUpdateCmd(),
		newDeleteCmd(),
	)
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all package update policies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			resources, err := jsonapi.GetAllPages[packageUpdatePolicyAttrs](cmd.Context(), client, "/api/v1/package_update_policies")
			if err != nil {
				return err
			}
			var rows [][]string
			var items []PackageUpdatePolicy
			for _, r := range resources {
				p := policyFromResource(r)
				items = append(items, p)
				rows = append(rows, policyRow(p))
			}
			return renderer.Render(policyCols, rows, httpclient.Envelope[[]PackageUpdatePolicy]{Data: items})
		},
	}
}

func newCreateCmd() *cobra.Command {
	var deploymentID, packageID string
	var maxAttempts int
	var isBlocked bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new package update policy",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := packageUpdatePolicyWriteAttrs{
				DeploymentID: deploymentID,
				PackageID:    packageID,
			}
			if cmd.Flags().Changed("max-attempts") {
				v := maxAttempts
				attrs.MaxAttempts = &v
			}
			if cmd.Flags().Changed("is-blocked") {
				v := isBlocked
				attrs.IsBlocked = &v
			}

			body := jsonapi.Wrap(policyResourceType, attrs)
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			res, err := httpclient.PostJSONAPISingle[packageUpdatePolicyAttrs](cmd.Context(), client, "/api/v1/package_update_policies", body)
			if err != nil {
				return err
			}
			p := policyFromResource(res)
			return renderer.Render(policyCols, [][]string{policyRow(p)}, httpclient.Envelope[PackageUpdatePolicy]{Data: p})
		},
	}
	cmd.Flags().StringVar(&deploymentID, "deployment-id", "", "Deployment ID")
	cmd.Flags().StringVar(&packageID, "package-id", "", "Package ID")
	cmd.Flags().IntVar(&maxAttempts, "max-attempts", 0, "Maximum update attempts")
	cmd.Flags().BoolVar(&isBlocked, "is-blocked", false, "Block automatic updates")
	if err := cmd.MarkFlagRequired("deployment-id"); err != nil {
		panic(err)
	}
	return cmd
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a package update policy by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/package_update_policies/" + url.PathEscape(args[0])
			res, err := jsonapi.GetSingle[packageUpdatePolicyAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			p := policyFromResource(res.Resource)
			return renderer.Render(policyCols, [][]string{policyRow(p)}, httpclient.Envelope[PackageUpdatePolicy]{Data: p})
		},
	}
}

func newUpdateCmd() *cobra.Command {
	var deploymentID, packageID string
	var maxAttempts int
	var isBlocked bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a package update policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := packageUpdatePolicyWriteAttrs{}
			anyChanged := false
			if cmd.Flags().Changed("deployment-id") {
				attrs.DeploymentID = deploymentID
				anyChanged = true
			}
			if cmd.Flags().Changed("package-id") {
				attrs.PackageID = packageID
				anyChanged = true
			}
			if cmd.Flags().Changed("max-attempts") {
				v := maxAttempts
				attrs.MaxAttempts = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("is-blocked") {
				v := isBlocked
				attrs.IsBlocked = &v
				anyChanged = true
			}
			if !anyChanged {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			body := jsonapi.Wrap(policyResourceType, attrs)
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			initialPath := "/api/v1/package_update_policies/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[packageUpdatePolicyAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			selfPath := jsonapi.SelfPath(fetched.SelfLink, initialPath)
			res, err := jsonapi.PatchSingle[packageUpdatePolicyAttrs](cmd.Context(), client, selfPath, body)
			if err != nil {
				return err
			}
			p := policyFromResource(res)
			return renderer.Render(policyCols, [][]string{policyRow(p)}, httpclient.Envelope[PackageUpdatePolicy]{Data: p})
		},
	}
	cmd.Flags().StringVar(&deploymentID, "deployment-id", "", "Deployment ID")
	cmd.Flags().StringVar(&packageID, "package-id", "", "Package ID")
	cmd.Flags().IntVar(&maxAttempts, "max-attempts", 0, "Maximum update attempts")
	cmd.Flags().BoolVar(&isBlocked, "is-blocked", false, "Block automatic updates")
	return cmd
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a package update policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := cmd.OutOrStdout().Write([]byte("DELETE /api/v1/package_update_policies/" + url.PathEscape(args[0]) + "\n"))
				return err
			}
			initialPath := "/api/v1/package_update_policies/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[packageUpdatePolicyAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			return client.Delete(cmd.Context(), jsonapi.SelfPath(fetched.SelfLink, initialPath))
		},
	}
}
