// Package deployments provides the "deployments" subcommand tree for clusterctl.
package deployments

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

const deploymentResourceType = "deployments"

// Deployment is the API response shape for a deployment resource.
type Deployment struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	ClusterID      string `json:"cluster_id,omitempty"`
	PackageName    string `json:"package_name,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	ValuesOverride string `json:"values_override,omitempty"`
	Status         string `json:"status,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

type deploymentAttrs struct {
	Name           string `json:"name,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	ClusterID      string `json:"cluster_id,omitempty"`
	PackageName    string `json:"package_name,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	ValuesOverride string `json:"values_override,omitempty"`
	Status         string `json:"status,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

func deploymentFromResource(r httpclient.Resource[deploymentAttrs]) Deployment {
	a := r.Attributes
	return Deployment{
		ID:             r.ID,
		Name:           a.Name,
		Namespace:      a.Namespace,
		ProjectID:      a.ProjectID,
		ClusterID:      a.ClusterID,
		PackageName:    a.PackageName,
		PackageVersion: a.PackageVersion,
		ValuesOverride: a.ValuesOverride,
		Status:         a.Status,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

// createDeploymentAttrs matches DeploymentRequest.data.attributes in the spec.
type createDeploymentAttrs struct {
	ProjectID      string `json:"project_id"`
	ClusterID      string `json:"cluster_id,omitempty"`
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	PackageName    string `json:"package_name"`
	PackageVersion string `json:"package_version"`
	ValuesOverride string `json:"values_override,omitempty"`
}

var deploymentCols = []output.Column{
	{Header: "ID"},
	{Header: "NAME"},
	{Header: "PROJECT"},
	{Header: "CLUSTER"},
	{Header: "PACKAGE"},
	{Header: "STATUS"},
}

func deploymentRow(d Deployment) []string {
	return []string{d.ID, d.Name, d.ProjectID, d.ClusterID, d.PackageName, d.Status}
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
			resources, err := jsonapi.GetAllPages[deploymentAttrs](cmd.Context(), client, "/api/v1/deployments")
			if err != nil {
				return err
			}
			var rows [][]string
			var items []Deployment
			for _, r := range resources {
				d := deploymentFromResource(r)
				items = append(items, d)
				rows = append(rows, deploymentRow(d))
			}
			return renderer.Render(deploymentCols, rows, httpclient.Envelope[[]Deployment]{Data: items})
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
			res, err := jsonapi.GetSingle[deploymentAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			d := deploymentFromResource(res.Resource)
			return renderer.Render(deploymentCols, [][]string{deploymentRow(d)}, httpclient.Envelope[Deployment]{Data: d})
		},
	}
}

func newCreateCmd() *cobra.Command {
	var name, namespace, projectID, clusterID, packageName, packageVersion, valuesOverride string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new deployment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := jsonapi.Wrap(deploymentResourceType, createDeploymentAttrs{
				ProjectID:      projectID,
				ClusterID:      clusterID,
				Name:           name,
				Namespace:      namespace,
				PackageName:    packageName,
				PackageVersion: packageVersion,
				ValuesOverride: valuesOverride,
			})
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			res, err := httpclient.PostJSONAPISingle[deploymentAttrs](cmd.Context(), client, "/api/v1/deployments", body)
			if err != nil {
				return err
			}
			d := deploymentFromResource(res)
			return renderer.Render(deploymentCols, [][]string{deploymentRow(d)}, httpclient.Envelope[Deployment]{Data: d})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Deployment name")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "Cluster ID")
	cmd.Flags().StringVar(&packageName, "package-name", "", "Package name")
	cmd.Flags().StringVar(&packageVersion, "package-version", "", "Package version")
	cmd.Flags().StringVar(&valuesOverride, "values-override", "", "Values override (YAML/JSON string)")
	if err := cmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("namespace"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("project-id"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("package-name"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("package-version"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var name, namespace, projectID, clusterID, packageName, packageVersion, valuesOverride string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := createDeploymentAttrs{}
			anyChanged := false
			if cmd.Flags().Changed("name") {
				attrs.Name = name
				anyChanged = true
			}
			if cmd.Flags().Changed("namespace") {
				attrs.Namespace = namespace
				anyChanged = true
			}
			if cmd.Flags().Changed("project-id") {
				attrs.ProjectID = projectID
				anyChanged = true
			}
			if cmd.Flags().Changed("cluster-id") {
				attrs.ClusterID = clusterID
				anyChanged = true
			}
			if cmd.Flags().Changed("package-name") {
				attrs.PackageName = packageName
				anyChanged = true
			}
			if cmd.Flags().Changed("package-version") {
				attrs.PackageVersion = packageVersion
				anyChanged = true
			}
			if cmd.Flags().Changed("values-override") {
				attrs.ValuesOverride = valuesOverride
				anyChanged = true
			}
			if !anyChanged {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			body := jsonapi.Wrap(deploymentResourceType, attrs)
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			initialPath := "/api/v1/deployments/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[deploymentAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			selfPath := jsonapi.SelfPath(fetched.SelfLink, initialPath)
			res, err := jsonapi.PatchSingle[deploymentAttrs](cmd.Context(), client, selfPath, body)
			if err != nil {
				return err
			}
			d := deploymentFromResource(res)
			return renderer.Render(deploymentCols, [][]string{deploymentRow(d)}, httpclient.Envelope[Deployment]{Data: d})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Deployment name")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "Cluster ID")
	cmd.Flags().StringVar(&packageName, "package-name", "", "Package name")
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
			initialPath := "/api/v1/deployments/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[deploymentAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			return client.Delete(cmd.Context(), jsonapi.SelfPath(fetched.SelfLink, initialPath))
		},
	}
}
