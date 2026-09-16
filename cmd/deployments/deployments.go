// Package deployments provides the "deployments" subcommand tree for clusterctl.
package deployments

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

const deploymentResourceType = "deployments"

// Toleration models a Kubernetes toleration entry for the tolerations field.
type Toleration struct {
	Key      string `json:"key,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
	Effect   string `json:"effect,omitempty"`
}

// Deployment is the API response shape for a deployment resource.
type Deployment struct {
	ID                      string            `json:"id"`
	Name                    string            `json:"name,omitempty"`
	Namespace               string            `json:"namespace,omitempty"`
	ProjectID               string            `json:"project_id,omitempty"`
	ClusterID               string            `json:"cluster_id,omitempty"`
	PackageName             string            `json:"package_name,omitempty"`
	PackageVersion          string            `json:"package_version,omitempty"`
	EnvironmentPreset       string            `json:"environment_preset,omitempty"`
	ValuesOverride          string            `json:"values_override,omitempty"`
	Status                  string            `json:"status,omitempty"`
	IsAI                    *bool             `json:"is_ai,omitempty"`
	HPAEnabled              *bool             `json:"hpa_enabled,omitempty"`
	ScalingMode             string            `json:"scaling_mode,omitempty"`
	MinReplicas             *int              `json:"min_replicas,omitempty"`
	MaxReplicas             *int              `json:"max_replicas,omitempty"`
	DesiredReplicas         *int              `json:"desired_replicas,omitempty"`
	CPUTargetUtilization    *int              `json:"cpu_target_utilization,omitempty"`
	MemoryTargetUtilization *int              `json:"memory_target_utilization,omitempty"`
	CPURequest              string            `json:"cpu_request,omitempty"`
	CPULimit                string            `json:"cpu_limit,omitempty"`
	MemoryRequest           string            `json:"memory_request,omitempty"`
	MemoryLimit             string            `json:"memory_limit,omitempty"`
	ScalingProfile          string            `json:"scaling_profile,omitempty"`
	PlacementPolicy         string            `json:"placement_policy,omitempty"`
	CanaryEnabled           *bool             `json:"canary_enabled,omitempty"`
	CanaryStepWeight        *int              `json:"canary_step_weight,omitempty"`
	CanaryInterval          string            `json:"canary_interval,omitempty"`
	CanaryMaxWeight         *int              `json:"canary_max_weight,omitempty"`
	CanarySuccessThreshold  *float64          `json:"canary_success_threshold,omitempty"`
	CanaryErrorThreshold    *float64          `json:"canary_error_threshold,omitempty"`
	CanaryLatencyP99ms      *int              `json:"canary_latency_p99_ms,omitempty"`
	NodeSelector            map[string]string `json:"node_selector,omitempty"`
	Tolerations             []Toleration      `json:"tolerations,omitempty"`
	TemplateExtraResources  map[string]string `json:"template_extra_resources,omitempty"`
	GitopsSyncStatus        string            `json:"gitops_sync_status,omitempty"`
	GitopsSourceRepo        string            `json:"gitops_source_repo,omitempty"`
	GitopsBasePath          string            `json:"gitops_base_path,omitempty"`
	GitopsCurrentRevision   string            `json:"gitops_current_revision,omitempty"`
	GitopsLastReconciledAt  string            `json:"gitops_last_reconciled_at,omitempty"`
	GitopsGithubFilesURL    string            `json:"gitops_github_files_url,omitempty"`
	PublishError            string            `json:"publish_error,omitempty"`
	IsAutoBlocked           *bool             `json:"is_auto_blocked,omitempty"`
	IsPinned                *bool             `json:"is_pinned,omitempty"`
	CreatedAt               string            `json:"created_at,omitempty"`
	UpdatedAt               string            `json:"updated_at,omitempty"`
}

type deploymentAttrs struct {
	Name                    string            `json:"name,omitempty"`
	Namespace               string            `json:"namespace,omitempty"`
	ProjectID               string            `json:"project_id,omitempty"`
	ClusterID               string            `json:"cluster_id,omitempty"`
	PackageName             string            `json:"package_name,omitempty"`
	PackageVersion          string            `json:"package_version,omitempty"`
	EnvironmentPreset       string            `json:"environment_preset,omitempty"`
	ValuesOverride          string            `json:"values_override,omitempty"`
	Status                  string            `json:"status,omitempty"`
	IsAI                    *bool             `json:"is_ai,omitempty"`
	HPAEnabled              *bool             `json:"hpa_enabled,omitempty"`
	ScalingMode             string            `json:"scaling_mode,omitempty"`
	MinReplicas             *int              `json:"min_replicas,omitempty"`
	MaxReplicas             *int              `json:"max_replicas,omitempty"`
	DesiredReplicas         *int              `json:"desired_replicas,omitempty"`
	CPUTargetUtilization    *int              `json:"cpu_target_utilization,omitempty"`
	MemoryTargetUtilization *int              `json:"memory_target_utilization,omitempty"`
	CPURequest              string            `json:"cpu_request,omitempty"`
	CPULimit                string            `json:"cpu_limit,omitempty"`
	MemoryRequest           string            `json:"memory_request,omitempty"`
	MemoryLimit             string            `json:"memory_limit,omitempty"`
	ScalingProfile          string            `json:"scaling_profile,omitempty"`
	PlacementPolicy         string            `json:"placement_policy,omitempty"`
	CanaryEnabled           *bool             `json:"canary_enabled,omitempty"`
	CanaryStepWeight        *int              `json:"canary_step_weight,omitempty"`
	CanaryInterval          string            `json:"canary_interval,omitempty"`
	CanaryMaxWeight         *int              `json:"canary_max_weight,omitempty"`
	CanarySuccessThreshold  *float64          `json:"canary_success_threshold,omitempty"`
	CanaryErrorThreshold    *float64          `json:"canary_error_threshold,omitempty"`
	CanaryLatencyP99ms      *int              `json:"canary_latency_p99_ms,omitempty"`
	NodeSelector            map[string]string `json:"node_selector,omitempty"`
	Tolerations             []Toleration      `json:"tolerations,omitempty"`
	TemplateExtraResources  map[string]string `json:"template_extra_resources,omitempty"`
	GitopsSyncStatus        string            `json:"gitops_sync_status,omitempty"`
	GitopsSourceRepo        string            `json:"gitops_source_repo,omitempty"`
	GitopsBasePath          string            `json:"gitops_base_path,omitempty"`
	GitopsCurrentRevision   string            `json:"gitops_current_revision,omitempty"`
	GitopsLastReconciledAt  string            `json:"gitops_last_reconciled_at,omitempty"`
	GitopsGithubFilesURL    string            `json:"gitops_github_files_url,omitempty"`
	PublishError            string            `json:"publish_error,omitempty"`
	IsAutoBlocked           *bool             `json:"is_auto_blocked,omitempty"`
	IsPinned                *bool             `json:"is_pinned,omitempty"`
	CreatedAt               string            `json:"created_at,omitempty"`
	UpdatedAt               string            `json:"updated_at,omitempty"`
}

func deploymentFromResource(r httpclient.Resource[deploymentAttrs]) Deployment {
	a := r.Attributes
	return Deployment{
		ID:                      r.ID,
		Name:                    a.Name,
		Namespace:               a.Namespace,
		ProjectID:               a.ProjectID,
		ClusterID:               a.ClusterID,
		PackageName:             a.PackageName,
		PackageVersion:          a.PackageVersion,
		EnvironmentPreset:       a.EnvironmentPreset,
		ValuesOverride:          a.ValuesOverride,
		Status:                  a.Status,
		IsAI:                    a.IsAI,
		HPAEnabled:              a.HPAEnabled,
		ScalingMode:             a.ScalingMode,
		MinReplicas:             a.MinReplicas,
		MaxReplicas:             a.MaxReplicas,
		DesiredReplicas:         a.DesiredReplicas,
		CPUTargetUtilization:    a.CPUTargetUtilization,
		MemoryTargetUtilization: a.MemoryTargetUtilization,
		CPURequest:              a.CPURequest,
		CPULimit:                a.CPULimit,
		MemoryRequest:           a.MemoryRequest,
		MemoryLimit:             a.MemoryLimit,
		ScalingProfile:          a.ScalingProfile,
		PlacementPolicy:         a.PlacementPolicy,
		CanaryEnabled:           a.CanaryEnabled,
		CanaryStepWeight:        a.CanaryStepWeight,
		CanaryInterval:          a.CanaryInterval,
		CanaryMaxWeight:         a.CanaryMaxWeight,
		CanarySuccessThreshold:  a.CanarySuccessThreshold,
		CanaryErrorThreshold:    a.CanaryErrorThreshold,
		CanaryLatencyP99ms:      a.CanaryLatencyP99ms,
		NodeSelector:            a.NodeSelector,
		Tolerations:             a.Tolerations,
		TemplateExtraResources:  a.TemplateExtraResources,
		GitopsSyncStatus:        a.GitopsSyncStatus,
		GitopsSourceRepo:        a.GitopsSourceRepo,
		GitopsBasePath:          a.GitopsBasePath,
		GitopsCurrentRevision:   a.GitopsCurrentRevision,
		GitopsLastReconciledAt:  a.GitopsLastReconciledAt,
		GitopsGithubFilesURL:    a.GitopsGithubFilesURL,
		PublishError:            a.PublishError,
		IsAutoBlocked:           a.IsAutoBlocked,
		IsPinned:                a.IsPinned,
		CreatedAt:               a.CreatedAt,
		UpdatedAt:               a.UpdatedAt,
	}
}

// UpdateRun is the API response shape for an update_runs resource.
type UpdateRun struct {
	ID             string `json:"id"`
	DeploymentID   string `json:"deployment_id,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	Status         string `json:"status,omitempty"`
	Attempt        int    `json:"attempt,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
}

type updateRunAttrs struct {
	DeploymentID   string `json:"deployment_id,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	Status         string `json:"status,omitempty"`
	Attempt        int    `json:"attempt,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
}

// Pin is the API response shape for a pins resource.
type Pin struct {
	ID             string `json:"id"`
	DeploymentID   string `json:"deployment_id,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	IsAutoBlocked  *bool  `json:"is_auto_blocked,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

type pinAttrs struct {
	DeploymentID   string `json:"deployment_id,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	IsAutoBlocked  *bool  `json:"is_auto_blocked,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// PackageUpdate is the API response shape for a package_updates resource.
type PackageUpdate struct {
	ID           string `json:"id"`
	DeploymentID string `json:"deployment_id,omitempty"`
	FromVersion  string `json:"from_version,omitempty"`
	ToVersion    string `json:"to_version,omitempty"`
	Status       string `json:"status,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

type packageUpdateAttrs struct {
	DeploymentID string `json:"deployment_id,omitempty"`
	FromVersion  string `json:"from_version,omitempty"`
	ToVersion    string `json:"to_version,omitempty"`
	Status       string `json:"status,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// Rollout is the API response shape for a rollouts resource.
type Rollout struct {
	ID           string `json:"id"`
	DeploymentID string `json:"deployment_id,omitempty"`
	Status       string `json:"status,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

type rolloutAttrs struct {
	DeploymentID string `json:"deployment_id,omitempty"`
	Status       string `json:"status,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// deploymentWriteAttrs matches DeploymentRequest.data.attributes in the spec.
type deploymentWriteAttrs struct {
	ProjectID               string            `json:"project_id,omitempty"`
	ClusterID               string            `json:"cluster_id,omitempty"`
	Name                    string            `json:"name,omitempty"`
	Namespace               string            `json:"namespace,omitempty"`
	PackageName             string            `json:"package_name,omitempty"`
	PackageVersion          string            `json:"package_version,omitempty"`
	EnvironmentPreset       string            `json:"environment_preset,omitempty"`
	ValuesOverride          string            `json:"values_override,omitempty"`
	IsAI                    *bool             `json:"is_ai,omitempty"`
	ScalingMode             string            `json:"scaling_mode,omitempty"`
	MinReplicas             *int              `json:"min_replicas,omitempty"`
	MaxReplicas             *int              `json:"max_replicas,omitempty"`
	DesiredReplicas         *int              `json:"desired_replicas,omitempty"`
	CPUTargetUtilization    *int              `json:"cpu_target_utilization,omitempty"`
	MemoryTargetUtilization *int              `json:"memory_target_utilization,omitempty"`
	CPURequest              string            `json:"cpu_request,omitempty"`
	CPULimit                string            `json:"cpu_limit,omitempty"`
	MemoryRequest           string            `json:"memory_request,omitempty"`
	MemoryLimit             string            `json:"memory_limit,omitempty"`
	ScalingProfile          string            `json:"scaling_profile,omitempty"`
	PlacementPolicy         string            `json:"placement_policy,omitempty"`
	CanaryEnabled           *bool             `json:"canary_enabled,omitempty"`
	CanaryStepWeight        *int              `json:"canary_step_weight,omitempty"`
	CanaryInterval          string            `json:"canary_interval,omitempty"`
	CanaryMaxWeight         *int              `json:"canary_max_weight,omitempty"`
	CanarySuccessThreshold  *float64          `json:"canary_success_threshold,omitempty"`
	CanaryErrorThreshold    *float64          `json:"canary_error_threshold,omitempty"`
	CanaryLatencyP99ms      *int              `json:"canary_latency_p99_ms,omitempty"`
	NodeSelector            map[string]string `json:"node_selector,omitempty"`
	Tolerations             []Toleration      `json:"tolerations,omitempty"`
	TemplateExtraResources  map[string]string `json:"template_extra_resources,omitempty"`
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
		newUpdateRunsCmd(),
		newPinCmd(),
		newUnpinCmd(),
		newPackageUpdateCmd(),
		newRolloutCmd(),
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

func parseNodeSelector(s string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("--node-selector: invalid JSON object: %w", err)
	}
	return m, nil
}

func parseTolerations(s string) ([]Toleration, error) {
	var t []Toleration
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return nil, fmt.Errorf("--tolerations: invalid JSON array: %w", err)
	}
	return t, nil
}

func parseTemplateExtraResources(s string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("--template-extra-resources: invalid JSON object (must be a filename→YAML-content map): %w", err)
	}
	return m, nil
}

func newCreateCmd() *cobra.Command {
	var name, namespace, projectID, clusterID, packageName, packageVersion, valuesOverride string
	var environmentPreset, scalingMode, scalingProfile, placementPolicy string
	var canaryInterval, cpuRequest, cpuLimit, memoryRequest, memoryLimit string
	var nodeSelectorJSON, tolerationsJSON, templateExtraResourcesJSON string
	var isAI, canaryEnabled bool
	var minReplicas, maxReplicas, desiredReplicas int
	var cpuTargetUtilization, memoryTargetUtilization int
	var canaryStepWeight, canaryMaxWeight, canaryLatencyP99ms int
	var canarySuccessThreshold, canaryErrorThreshold float64

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new deployment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := deploymentWriteAttrs{
				ProjectID:      projectID,
				ClusterID:      clusterID,
				Name:           name,
				Namespace:      namespace,
				PackageName:    packageName,
				PackageVersion: packageVersion,
				ValuesOverride: valuesOverride,
			}

			if cmd.Flags().Changed("environment-preset") {
				attrs.EnvironmentPreset = environmentPreset
			}
			if cmd.Flags().Changed("scaling-mode") {
				attrs.ScalingMode = scalingMode
			}
			if cmd.Flags().Changed("scaling-profile") {
				attrs.ScalingProfile = scalingProfile
			}
			if cmd.Flags().Changed("placement-policy") {
				attrs.PlacementPolicy = placementPolicy
			}
			if cmd.Flags().Changed("canary-interval") {
				attrs.CanaryInterval = canaryInterval
			}
			if cmd.Flags().Changed("cpu-request") {
				attrs.CPURequest = cpuRequest
			}
			if cmd.Flags().Changed("cpu-limit") {
				attrs.CPULimit = cpuLimit
			}
			if cmd.Flags().Changed("memory-request") {
				attrs.MemoryRequest = memoryRequest
			}
			if cmd.Flags().Changed("memory-limit") {
				attrs.MemoryLimit = memoryLimit
			}
			if cmd.Flags().Changed("is-ai") {
				v := isAI
				attrs.IsAI = &v
			}
			if cmd.Flags().Changed("canary-enabled") {
				v := canaryEnabled
				attrs.CanaryEnabled = &v
			}
			if cmd.Flags().Changed("min-replicas") {
				v := minReplicas
				attrs.MinReplicas = &v
			}
			if cmd.Flags().Changed("max-replicas") {
				v := maxReplicas
				attrs.MaxReplicas = &v
			}
			if cmd.Flags().Changed("desired-replicas") {
				v := desiredReplicas
				attrs.DesiredReplicas = &v
			}
			if cmd.Flags().Changed("cpu-target-utilization") {
				v := cpuTargetUtilization
				attrs.CPUTargetUtilization = &v
			}
			if cmd.Flags().Changed("memory-target-utilization") {
				v := memoryTargetUtilization
				attrs.MemoryTargetUtilization = &v
			}
			if cmd.Flags().Changed("canary-step-weight") {
				v := canaryStepWeight
				attrs.CanaryStepWeight = &v
			}
			if cmd.Flags().Changed("canary-max-weight") {
				v := canaryMaxWeight
				attrs.CanaryMaxWeight = &v
			}
			if cmd.Flags().Changed("canary-latency-p99-ms") {
				v := canaryLatencyP99ms
				attrs.CanaryLatencyP99ms = &v
			}
			if cmd.Flags().Changed("canary-success-threshold") {
				v := canarySuccessThreshold
				attrs.CanarySuccessThreshold = &v
			}
			if cmd.Flags().Changed("canary-error-threshold") {
				v := canaryErrorThreshold
				attrs.CanaryErrorThreshold = &v
			}
			if cmd.Flags().Changed("node-selector") {
				ns, err := parseNodeSelector(nodeSelectorJSON)
				if err != nil {
					return err
				}
				attrs.NodeSelector = ns
			}
			if cmd.Flags().Changed("tolerations") {
				tols, err := parseTolerations(tolerationsJSON)
				if err != nil {
					return err
				}
				attrs.Tolerations = tols
			}
			if cmd.Flags().Changed("template-extra-resources") {
				ter, err := parseTemplateExtraResources(templateExtraResourcesJSON)
				if err != nil {
					return err
				}
				attrs.TemplateExtraResources = ter
			}

			body := jsonapi.Wrap(deploymentResourceType, attrs)
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
	cmd.Flags().StringVar(&environmentPreset, "environment-preset", "", "Environment preset name")
	cmd.Flags().StringVar(&scalingMode, "scaling-mode", "", `Scaling mode (e.g. "hpa", "keda", "manual")`)
	cmd.Flags().StringVar(&scalingProfile, "scaling-profile", "", "Scaling profile name")
	cmd.Flags().StringVar(&placementPolicy, "placement-policy", "", "Pod placement policy name")
	cmd.Flags().StringVar(&canaryInterval, "canary-interval", "", `Canary analysis interval (e.g. "1m")`)
	cmd.Flags().StringVar(&cpuRequest, "cpu-request", "", `CPU resource request (e.g. "250m")`)
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", `CPU resource limit (e.g. "500m")`)
	cmd.Flags().StringVar(&memoryRequest, "memory-request", "", `Memory resource request (e.g. "256Mi")`)
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", `Memory resource limit (e.g. "512Mi")`)
	cmd.Flags().StringVar(&nodeSelectorJSON, "node-selector", "", `Node selector as a JSON object (e.g. '{"kubernetes.io/os":"linux"}')`)
	cmd.Flags().StringVar(&tolerationsJSON, "tolerations", "", `Tolerations as a JSON array (e.g. '[{"key":"foo","operator":"Equal","value":"bar","effect":"NoSchedule"}]')`)
	cmd.Flags().StringVar(&templateExtraResourcesJSON, "template-extra-resources", "", "Extra template files as a JSON object mapping filename to YAML content")
	cmd.Flags().BoolVar(&isAI, "is-ai", false, "Mark as an AI workload (enables AI-specific defaults)")
	cmd.Flags().BoolVar(&canaryEnabled, "canary-enabled", false, "Enable canary rollout via Flagger")
	cmd.Flags().IntVar(&minReplicas, "min-replicas", 0, "Minimum replica count")
	cmd.Flags().IntVar(&maxReplicas, "max-replicas", 0, "Maximum replica count")
	cmd.Flags().IntVar(&desiredReplicas, "desired-replicas", 0, "Desired static replica count (used when scaling-mode is manual)")
	cmd.Flags().IntVar(&cpuTargetUtilization, "cpu-target-utilization", 0, "CPU HPA target utilization percentage (1-100)")
	cmd.Flags().IntVar(&memoryTargetUtilization, "memory-target-utilization", 0, "Memory HPA target utilization percentage (1-100)")
	cmd.Flags().IntVar(&canaryStepWeight, "canary-step-weight", 0, "Canary step weight percentage per step")
	cmd.Flags().IntVar(&canaryMaxWeight, "canary-max-weight", 0, "Maximum canary traffic weight percentage")
	cmd.Flags().IntVar(&canaryLatencyP99ms, "canary-latency-p99-ms", 0, "Canary p99 latency threshold in ms")
	cmd.Flags().Float64Var(&canarySuccessThreshold, "canary-success-threshold", 0, "Canary success rate threshold (0-1)")
	cmd.Flags().Float64Var(&canaryErrorThreshold, "canary-error-threshold", 0, "Canary error rate threshold (0-1)")
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
	var environmentPreset, scalingMode, scalingProfile, placementPolicy string
	var canaryInterval, cpuRequest, cpuLimit, memoryRequest, memoryLimit string
	var nodeSelectorJSON, tolerationsJSON, templateExtraResourcesJSON string
	var isAI, canaryEnabled bool
	var minReplicas, maxReplicas, desiredReplicas int
	var cpuTargetUtilization, memoryTargetUtilization int
	var canaryStepWeight, canaryMaxWeight, canaryLatencyP99ms int
	var canarySuccessThreshold, canaryErrorThreshold float64

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := deploymentWriteAttrs{}
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
			if cmd.Flags().Changed("environment-preset") {
				attrs.EnvironmentPreset = environmentPreset
				anyChanged = true
			}
			if cmd.Flags().Changed("scaling-mode") {
				attrs.ScalingMode = scalingMode
				anyChanged = true
			}
			if cmd.Flags().Changed("scaling-profile") {
				attrs.ScalingProfile = scalingProfile
				anyChanged = true
			}
			if cmd.Flags().Changed("placement-policy") {
				attrs.PlacementPolicy = placementPolicy
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-interval") {
				attrs.CanaryInterval = canaryInterval
				anyChanged = true
			}
			if cmd.Flags().Changed("cpu-request") {
				attrs.CPURequest = cpuRequest
				anyChanged = true
			}
			if cmd.Flags().Changed("cpu-limit") {
				attrs.CPULimit = cpuLimit
				anyChanged = true
			}
			if cmd.Flags().Changed("memory-request") {
				attrs.MemoryRequest = memoryRequest
				anyChanged = true
			}
			if cmd.Flags().Changed("memory-limit") {
				attrs.MemoryLimit = memoryLimit
				anyChanged = true
			}
			if cmd.Flags().Changed("is-ai") {
				v := isAI
				attrs.IsAI = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-enabled") {
				v := canaryEnabled
				attrs.CanaryEnabled = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("min-replicas") {
				v := minReplicas
				attrs.MinReplicas = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("max-replicas") {
				v := maxReplicas
				attrs.MaxReplicas = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("desired-replicas") {
				v := desiredReplicas
				attrs.DesiredReplicas = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("cpu-target-utilization") {
				v := cpuTargetUtilization
				attrs.CPUTargetUtilization = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("memory-target-utilization") {
				v := memoryTargetUtilization
				attrs.MemoryTargetUtilization = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-step-weight") {
				v := canaryStepWeight
				attrs.CanaryStepWeight = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-max-weight") {
				v := canaryMaxWeight
				attrs.CanaryMaxWeight = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-latency-p99-ms") {
				v := canaryLatencyP99ms
				attrs.CanaryLatencyP99ms = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-success-threshold") {
				v := canarySuccessThreshold
				attrs.CanarySuccessThreshold = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-error-threshold") {
				v := canaryErrorThreshold
				attrs.CanaryErrorThreshold = &v
				anyChanged = true
			}
			if cmd.Flags().Changed("node-selector") {
				ns, err := parseNodeSelector(nodeSelectorJSON)
				if err != nil {
					return err
				}
				attrs.NodeSelector = ns
				anyChanged = true
			}
			if cmd.Flags().Changed("tolerations") {
				tols, err := parseTolerations(tolerationsJSON)
				if err != nil {
					return err
				}
				attrs.Tolerations = tols
				anyChanged = true
			}
			if cmd.Flags().Changed("template-extra-resources") {
				ter, err := parseTemplateExtraResources(templateExtraResourcesJSON)
				if err != nil {
					return err
				}
				attrs.TemplateExtraResources = ter
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
	cmd.Flags().StringVar(&environmentPreset, "environment-preset", "", "Environment preset name")
	cmd.Flags().StringVar(&scalingMode, "scaling-mode", "", `Scaling mode (e.g. "hpa", "keda", "manual")`)
	cmd.Flags().StringVar(&scalingProfile, "scaling-profile", "", "Scaling profile name")
	cmd.Flags().StringVar(&placementPolicy, "placement-policy", "", "Pod placement policy name")
	cmd.Flags().StringVar(&canaryInterval, "canary-interval", "", `Canary analysis interval (e.g. "1m")`)
	cmd.Flags().StringVar(&cpuRequest, "cpu-request", "", `CPU resource request (e.g. "250m")`)
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", `CPU resource limit (e.g. "500m")`)
	cmd.Flags().StringVar(&memoryRequest, "memory-request", "", `Memory resource request (e.g. "256Mi")`)
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", `Memory resource limit (e.g. "512Mi")`)
	cmd.Flags().StringVar(&nodeSelectorJSON, "node-selector", "", `Node selector as a JSON object (e.g. '{"kubernetes.io/os":"linux"}')`)
	cmd.Flags().StringVar(&tolerationsJSON, "tolerations", "", `Tolerations as a JSON array (e.g. '[{"key":"foo","operator":"Equal","value":"bar","effect":"NoSchedule"}]')`)
	cmd.Flags().StringVar(&templateExtraResourcesJSON, "template-extra-resources", "", "Extra template files as a JSON object mapping filename to YAML content")
	cmd.Flags().BoolVar(&isAI, "is-ai", false, "Mark as an AI workload (enables AI-specific defaults)")
	cmd.Flags().BoolVar(&canaryEnabled, "canary-enabled", false, "Enable canary rollout via Flagger")
	cmd.Flags().IntVar(&minReplicas, "min-replicas", 0, "Minimum replica count")
	cmd.Flags().IntVar(&maxReplicas, "max-replicas", 0, "Maximum replica count")
	cmd.Flags().IntVar(&desiredReplicas, "desired-replicas", 0, "Desired static replica count (used when scaling-mode is manual)")
	cmd.Flags().IntVar(&cpuTargetUtilization, "cpu-target-utilization", 0, "CPU HPA target utilization percentage (1-100)")
	cmd.Flags().IntVar(&memoryTargetUtilization, "memory-target-utilization", 0, "Memory HPA target utilization percentage (1-100)")
	cmd.Flags().IntVar(&canaryStepWeight, "canary-step-weight", 0, "Canary step weight percentage per step")
	cmd.Flags().IntVar(&canaryMaxWeight, "canary-max-weight", 0, "Maximum canary traffic weight percentage")
	cmd.Flags().IntVar(&canaryLatencyP99ms, "canary-latency-p99-ms", 0, "Canary p99 latency threshold in ms")
	cmd.Flags().Float64Var(&canarySuccessThreshold, "canary-success-threshold", 0, "Canary success rate threshold (0-1)")
	cmd.Flags().Float64Var(&canaryErrorThreshold, "canary-error-threshold", 0, "Canary error rate threshold (0-1)")
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

func newUpdateRunsCmd() *cobra.Command {
	updateRunCols := []output.Column{
		{Header: "ID"},
		{Header: "DEPLOYMENT_ID"},
		{Header: "STATUS"},
		{Header: "ATTEMPT"},
	}
	cmd := &cobra.Command{
		Use:   "update-runs",
		Short: "Manage deployment update runs",
	}

	listCmd := &cobra.Command{
		Use:   "list <deployment_id>",
		Short: "List update runs for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/deployments/" + url.PathEscape(args[0]) + "/update_runs"
			resources, err := jsonapi.GetAllPages[updateRunAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			var rows [][]string
			var items []UpdateRun
			for _, r := range resources {
				ur := UpdateRun{
					ID:             r.ID,
					DeploymentID:   r.Attributes.DeploymentID,
					PackageVersion: r.Attributes.PackageVersion,
					Status:         r.Attributes.Status,
					Attempt:        r.Attributes.Attempt,
					ErrorMessage:   r.Attributes.ErrorMessage,
					StartedAt:      r.Attributes.StartedAt,
					CompletedAt:    r.Attributes.CompletedAt,
				}
				items = append(items, ur)
				rows = append(rows, []string{ur.ID, ur.DeploymentID, ur.Status, fmt.Sprintf("%d", ur.Attempt)})
			}
			return renderer.Render(updateRunCols, rows, httpclient.Envelope[[]UpdateRun]{Data: items})
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <deployment_id> <run_id>",
		Short: "Get a deployment update run by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/deployments/" + url.PathEscape(args[0]) + "/update_runs/" + url.PathEscape(args[1])
			res, err := jsonapi.GetSingle[updateRunAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			ur := UpdateRun{
				ID:             res.Resource.ID,
				DeploymentID:   res.Resource.Attributes.DeploymentID,
				PackageVersion: res.Resource.Attributes.PackageVersion,
				Status:         res.Resource.Attributes.Status,
				Attempt:        res.Resource.Attributes.Attempt,
				ErrorMessage:   res.Resource.Attributes.ErrorMessage,
				StartedAt:      res.Resource.Attributes.StartedAt,
				CompletedAt:    res.Resource.Attributes.CompletedAt,
			}
			return renderer.Render(updateRunCols, [][]string{{ur.ID, ur.DeploymentID, ur.Status, fmt.Sprintf("%d", ur.Attempt)}}, httpclient.Envelope[UpdateRun]{Data: ur})
		},
	}

	cmd.AddCommand(listCmd, getCmd)
	return cmd
}

func newPinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pin <deployment_id>",
		Short: "Pin a deployment to its current package version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "POST /api/v1/deployments/%s/pin\n", url.PathEscape(args[0]))
				return err
			}
			path := "/api/v1/deployments/" + url.PathEscape(args[0]) + "/pin"
			res, err := httpclient.PostJSONAPISingle[pinAttrs](cmd.Context(), client, path, nil)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Attributes)
		},
	}
}

func newUnpinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unpin <deployment_id>",
		Short: "Remove the pin from a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "DELETE /api/v1/deployments/%s/pin\n", url.PathEscape(args[0]))
				return err
			}
			path := "/api/v1/deployments/" + url.PathEscape(args[0]) + "/pin"
			return client.Delete(cmd.Context(), path)
		},
	}
}

func newPackageUpdateCmd() *cobra.Command {
	puCols := []output.Column{
		{Header: "ID"},
		{Header: "DEPLOYMENT_ID"},
		{Header: "FROM_VERSION"},
		{Header: "TO_VERSION"},
		{Header: "STATUS"},
	}
	cmd := &cobra.Command{
		Use:   "package-update",
		Short: "Manage deployment package updates",
	}

	getCmd := &cobra.Command{
		Use:   "get <deployment_id>",
		Short: "Get the latest package update for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			path := "/api/v1/deployments/" + url.PathEscape(args[0]) + "/package_update"
			res, err := jsonapi.GetSingle[packageUpdateAttrs](cmd.Context(), client, path)
			if err != nil {
				return err
			}
			pu := PackageUpdate{
				ID:           res.Resource.ID,
				DeploymentID: res.Resource.Attributes.DeploymentID,
				FromVersion:  res.Resource.Attributes.FromVersion,
				ToVersion:    res.Resource.Attributes.ToVersion,
				Status:       res.Resource.Attributes.Status,
				CreatedAt:    res.Resource.Attributes.CreatedAt,
			}
			return renderer.Render(puCols, [][]string{{pu.ID, pu.DeploymentID, pu.FromVersion, pu.ToVersion, pu.Status}}, httpclient.Envelope[PackageUpdate]{Data: pu})
		},
	}

	applyCmd := &cobra.Command{
		Use:   "apply <deployment_id>",
		Short: "Apply a package update to a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "POST /api/v1/deployments/%s/package_update\n", url.PathEscape(args[0]))
				return err
			}
			path := "/api/v1/deployments/" + url.PathEscape(args[0]) + "/package_update"
			res, err := httpclient.PostJSONAPISingle[packageUpdateAttrs](cmd.Context(), client, path, nil)
			if err != nil {
				return err
			}
			pu := PackageUpdate{
				ID:           res.ID,
				DeploymentID: res.Attributes.DeploymentID,
				FromVersion:  res.Attributes.FromVersion,
				ToVersion:    res.Attributes.ToVersion,
				Status:       res.Attributes.Status,
				CreatedAt:    res.Attributes.CreatedAt,
			}
			return renderer.Render(puCols, [][]string{{pu.ID, pu.DeploymentID, pu.FromVersion, pu.ToVersion, pu.Status}}, httpclient.Envelope[PackageUpdate]{Data: pu})
		},
	}

	cmd.AddCommand(getCmd, applyCmd)
	return cmd
}

func newRolloutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollout <deployment_id>",
		Short: "Trigger a rollout for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())
			if gf.DryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "POST /api/v1/deployments/%s/rollout\n", url.PathEscape(args[0]))
				return err
			}
			path := "/api/v1/deployments/" + url.PathEscape(args[0]) + "/rollout"
			res, err := httpclient.PostJSONAPISingle[rolloutAttrs](cmd.Context(), client, path, nil)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Attributes)
		},
	}
}
