// Package clusters provides the "clusters" subcommand tree for clusterctl.
package clusters

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/output"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

const clusterResourceType = "clusters"

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

type clusterAttrs struct {
	Name            string `json:"name"`
	ClusterType     string `json:"cluster_type"`
	ParentClusterID string `json:"parent_cluster_id,omitempty"`
	Status          string `json:"status,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

func clusterFromResource(r httpclient.Resource[clusterAttrs]) Cluster {
	a := r.Attributes
	return Cluster{
		ID:              r.ID,
		Name:            a.Name,
		ClusterType:     a.ClusterType,
		ParentClusterID: a.ParentClusterID,
		Status:          a.Status,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}

// createClusterAttrs matches ClusterCreateRequest.data.attributes in the spec.
type createClusterAttrs struct {
	ClusterType               string `json:"cluster_type"`
	Name                      string `json:"name,omitempty"`
	ParentClusterID           string `json:"parent_cluster_id,omitempty"`
	Kubeconfig                string `json:"kubeconfig,omitempty"`
	GitopsRepoURL             string `json:"gitops_repo_url,omitempty"`
	K8sBaseHostname           string `json:"k8s_base_hostname,omitempty"`
	KubeconfigExportNamespace string `json:"kubeconfig_export_namespace,omitempty"`
	ClusterIssuerName         string `json:"cluster_issuer_name,omitempty"`
	IngressClassName          string `json:"ingress_class_name,omitempty"`
}

// updateClusterAttrs matches ClusterUpdateRequest.data.attributes in the spec.
type updateClusterAttrs struct {
	K8sBaseHostname           string `json:"k8s_base_hostname,omitempty"`
	KubeconfigExportNamespace string `json:"kubeconfig_export_namespace,omitempty"`
	ClusterIssuerName         string `json:"cluster_issuer_name,omitempty"`
	IngressClassName          string `json:"ingress_class_name,omitempty"`
	GitopsRepoURL             string `json:"gitops_repo_url,omitempty"`
	Kubeconfig                string `json:"kubeconfig,omitempty"`
}

// healthCheckAttrs matches HealthCheckResource.attributes in the spec.
type healthCheckAttrs struct {
	Status string `json:"status"`
}

// fluxBootstrapAttrs matches FluxBootstrapAttributes in the spec.
type fluxBootstrapAttrs struct {
	Name                string `json:"name,omitempty"`
	FluxBootstrapStatus string `json:"flux_bootstrap_status,omitempty"`
	FluxBootstrapError  string `json:"flux_bootstrap_error,omitempty"`
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
			resources, err := jsonapi.GetAllPages[clusterAttrs](cmd.Context(), client, "/api/v1/clusters")
			if err != nil {
				return err
			}
			var rows [][]string
			var items []Cluster
			for _, r := range resources {
				c := clusterFromResource(r)
				items = append(items, c)
				rows = append(rows, clusterRow(c))
			}
			return renderer.Render(clusterCols, rows, httpclient.Envelope[[]Cluster]{Data: items})
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
			res, err := jsonapi.GetSingle[clusterAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			c := clusterFromResource(res.Resource)
			return renderer.Render(clusterCols, [][]string{clusterRow(c)}, httpclient.Envelope[Cluster]{Data: c})
		},
	}
}

func newCreateCmd() *cobra.Command {
	var name, clusterType, parentClusterID string
	var kubeconfig, gitopsRepoURL, k8sBaseHostname string
	var kubeconfigExportNamespace, clusterIssuerName, ingressClassName string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new cluster",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			body := jsonapi.Wrap(clusterResourceType, createClusterAttrs{
				ClusterType:               clusterType,
				Name:                      name,
				ParentClusterID:           parentClusterID,
				Kubeconfig:                kubeconfig,
				GitopsRepoURL:             gitopsRepoURL,
				K8sBaseHostname:           k8sBaseHostname,
				KubeconfigExportNamespace: kubeconfigExportNamespace,
				ClusterIssuerName:         clusterIssuerName,
				IngressClassName:          ingressClassName,
			})
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			res, err := httpclient.PostJSONAPISingle[clusterAttrs](cmd.Context(), client, "/api/v1/clusters", body)
			if err != nil {
				return err
			}
			c := clusterFromResource(res)
			return renderer.Render(clusterCols, [][]string{clusterRow(c)}, httpclient.Envelope[Cluster]{Data: c})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Cluster name")
	cmd.Flags().StringVar(&clusterType, "cluster-type", "", "Cluster type (virtual|imported)")
	cmd.Flags().StringVar(&parentClusterID, "parent-cluster-id", "", "Parent cluster ID (for virtual clusters)")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig YAML")
	cmd.Flags().StringVar(&gitopsRepoURL, "gitops-repo-url", "", "GitOps repository URL")
	cmd.Flags().StringVar(&k8sBaseHostname, "k8s-base-hostname", "", "Kubernetes base hostname")
	cmd.Flags().StringVar(&kubeconfigExportNamespace, "kubeconfig-export-namespace", "", "Namespace for kubeconfig export")
	cmd.Flags().StringVar(&clusterIssuerName, "cluster-issuer-name", "", "Cluster issuer name")
	cmd.Flags().StringVar(&ingressClassName, "ingress-class-name", "", "Ingress class name")
	if err := cmd.MarkFlagRequired("cluster-type"); err != nil {
		panic(err)
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var k8sBaseHostname, kubeconfigExportNamespace, clusterIssuerName string
	var ingressClassName, gitopsRepoURL, kubeconfig string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := updateClusterAttrs{}
			anyChanged := false
			if cmd.Flags().Changed("k8s-base-hostname") {
				attrs.K8sBaseHostname = k8sBaseHostname
				anyChanged = true
			}
			if cmd.Flags().Changed("kubeconfig-export-namespace") {
				attrs.KubeconfigExportNamespace = kubeconfigExportNamespace
				anyChanged = true
			}
			if cmd.Flags().Changed("cluster-issuer-name") {
				attrs.ClusterIssuerName = clusterIssuerName
				anyChanged = true
			}
			if cmd.Flags().Changed("ingress-class-name") {
				attrs.IngressClassName = ingressClassName
				anyChanged = true
			}
			if cmd.Flags().Changed("gitops-repo-url") {
				attrs.GitopsRepoURL = gitopsRepoURL
				anyChanged = true
			}
			if cmd.Flags().Changed("kubeconfig") {
				attrs.Kubeconfig = kubeconfig
				anyChanged = true
			}
			if !anyChanged {
				return clierror.New(clierror.CodeUsage, "at least one flag required for update", "")
			}
			body := jsonapi.Wrap(clusterResourceType, attrs)
			if gf.DryRun {
				return output.JSONTo(cmd.OutOrStdout(), body)
			}
			initialPath := "/api/v1/clusters/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[clusterAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			selfPath := jsonapi.SelfPath(fetched.SelfLink, initialPath)
			res, err := jsonapi.PatchSingle[clusterAttrs](cmd.Context(), client, selfPath, body)
			if err != nil {
				return err
			}
			c := clusterFromResource(res)
			return renderer.Render(clusterCols, [][]string{clusterRow(c)}, httpclient.Envelope[Cluster]{Data: c})
		},
	}
	cmd.Flags().StringVar(&k8sBaseHostname, "k8s-base-hostname", "", "Kubernetes base hostname")
	cmd.Flags().StringVar(&kubeconfigExportNamespace, "kubeconfig-export-namespace", "", "Namespace for kubeconfig export")
	cmd.Flags().StringVar(&clusterIssuerName, "cluster-issuer-name", "", "Cluster issuer name")
	cmd.Flags().StringVar(&ingressClassName, "ingress-class-name", "", "Ingress class name")
	cmd.Flags().StringVar(&gitopsRepoURL, "gitops-repo-url", "", "GitOps repository URL")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig YAML")
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
			initialPath := "/api/v1/clusters/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[clusterAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			return client.Delete(cmd.Context(), jsonapi.SelfPath(fetched.SelfLink, initialPath))
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
			initialPath := "/api/v1/clusters/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[clusterAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			selfPath := jsonapi.SelfPath(fetched.SelfLink, initialPath)
			res, err := httpclient.PostJSONAPISingle[healthCheckAttrs](cmd.Context(), client, selfPath+"/health_check", nil)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Attributes)
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
			initialPath := "/api/v1/clusters/" + url.PathEscape(args[0])
			fetched, err := jsonapi.GetSingle[clusterAttrs](cmd.Context(), client, initialPath)
			if err != nil {
				return err
			}
			selfPath := jsonapi.SelfPath(fetched.SelfLink, initialPath)
			res, err := httpclient.PostJSONAPISingle[fluxBootstrapAttrs](cmd.Context(), client, selfPath+"/flux_bootstrap", nil)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Attributes)
		},
	}
}
