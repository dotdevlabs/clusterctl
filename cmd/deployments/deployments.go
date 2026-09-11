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

// tolerationEntry matches the toleration object shape in DeploymentRequest.
type tolerationEntry struct {
	Key      string `json:"key,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
	Effect   string `json:"effect,omitempty"`
}

// deploymentRequestAttrs matches DeploymentRequest.data.attributes in the spec.
// All optional fields use pointer types so omitempty distinguishes nil (never set)
// from zero/false (explicitly set by user).
type deploymentRequestAttrs struct {
	// required for create
	ProjectID      *string `json:"project_id,omitempty"`
	Name           *string `json:"name,omitempty"`
	Namespace      *string `json:"namespace,omitempty"`
	PackageName    *string `json:"package_name,omitempty"`
	PackageVersion *string `json:"package_version,omitempty"`
	// optional strings
	ClusterID         *string `json:"cluster_id,omitempty"`
	EnvironmentPreset *string `json:"environment_preset,omitempty"`
	ValuesOverride    *string `json:"values_override,omitempty"`
	ScalingMode       *string `json:"scaling_mode,omitempty"`
	CPURequest        *string `json:"cpu_request,omitempty"`
	CPULimit          *string `json:"cpu_limit,omitempty"`
	MemoryRequest     *string `json:"memory_request,omitempty"`
	MemoryLimit       *string `json:"memory_limit,omitempty"`
	ScalingProfile    *string `json:"scaling_profile,omitempty"`
	PlacementPolicy   *string `json:"placement_policy,omitempty"`
	CanaryInterval    *string `json:"canary_interval,omitempty"`
	// optional booleans
	IsAI          *bool `json:"is_ai,omitempty"`
	CanaryEnabled *bool `json:"canary_enabled,omitempty"`
	// optional integers
	MinReplicas             *int `json:"min_replicas,omitempty"`
	MaxReplicas             *int `json:"max_replicas,omitempty"`
	DesiredReplicas         *int `json:"desired_replicas,omitempty"`
	CPUTargetUtilization    *int `json:"cpu_target_utilization,omitempty"`
	MemoryTargetUtilization *int `json:"memory_target_utilization,omitempty"`
	CanaryStepWeight        *int `json:"canary_step_weight,omitempty"`
	CanaryMaxWeight         *int `json:"canary_max_weight,omitempty"`
	CanaryLatencyP99Ms      *int `json:"canary_latency_p99_ms,omitempty"`
	// optional floats
	CanarySuccessThreshold *float64 `json:"canary_success_threshold,omitempty"`
	CanaryErrorThreshold   *float64 `json:"canary_error_threshold,omitempty"`
	// optional structured objects (parsed from JSON flag values)
	NodeSelector           map[string]string      `json:"node_selector,omitempty"`
	Tolerations            []tolerationEntry      `json:"tolerations,omitempty"`
	TemplateValues         map[string]interface{} `json:"template_values,omitempty"`
	TemplateExtraResources map[string]string      `json:"template_extra_resources,omitempty"`
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

// parseNodeSelector parses a JSON object string into map[string]string.
func parseNodeSelector(s string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, clierror.New(clierror.CodeUsage, fmt.Sprintf("invalid --node-selector: must be a JSON object of string values: %v", err), "")
	}
	return m, nil
}

// parseTolerations parses a JSON array of toleration objects.
func parseTolerations(s string) ([]tolerationEntry, error) {
	var entries []tolerationEntry
	if err := json.Unmarshal([]byte(s), &entries); err != nil {
		return nil, clierror.New(clierror.CodeUsage, fmt.Sprintf("invalid --tolerations: must be a JSON array of toleration objects: %v", err), "")
	}
	return entries, nil
}

// parseTemplateValues parses a JSON object string into map[string]interface{}.
func parseTemplateValues(s string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, clierror.New(clierror.CodeUsage, fmt.Sprintf("invalid --template-values: must be a JSON object: %v", err), "")
	}
	return m, nil
}

// parseTemplateExtraResources parses a JSON object where all values must be strings.
func parseTemplateExtraResources(s string) (map[string]string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, clierror.New(clierror.CodeUsage, fmt.Sprintf("invalid --template-extra-resources: must be a JSON object: %v", err), "")
	}
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		sv, ok := v.(string)
		if !ok {
			return nil, clierror.New(clierror.CodeUsage, fmt.Sprintf("invalid --template-extra-resources: value for key %q must be a string, got %T", k, v), "")
		}
		result[k] = sv
	}
	return result, nil
}

// addDeploymentFlags registers all writable DeploymentRequest flags on cmd.
func addDeploymentFlags(cmd *cobra.Command,
	name, namespace, projectID, clusterID, packageName, packageVersion *string,
	environmentPreset, valuesOverride, scalingMode *string,
	cpuRequest, cpuLimit, memoryRequest, memoryLimit *string,
	scalingProfile, placementPolicy, canaryInterval *string,
	isAI, canaryEnabled *bool,
	minReplicas, maxReplicas, desiredReplicas *int,
	cpuTargetUtilization, memoryTargetUtilization *int,
	canaryStepWeight, canaryMaxWeight, canaryLatencyP99Ms *int,
	canarySuccessThreshold, canaryErrorThreshold *float64,
	nodeSelector, tolerations, templateValues, templateExtraResources *string,
) {
	f := cmd.Flags()
	f.StringVar(name, "name", "", "Deployment name")
	f.StringVar(namespace, "namespace", "", "Kubernetes namespace")
	f.StringVar(projectID, "project-id", "", "Project ID")
	f.StringVar(clusterID, "cluster-id", "", "Cluster ID")
	f.StringVar(packageName, "package-name", "", "Package name")
	f.StringVar(packageVersion, "package-version", "", "Package version")
	f.StringVar(environmentPreset, "environment-preset", "", "Environment preset name")
	f.StringVar(valuesOverride, "values-override", "", "Values override (YAML/JSON string)")
	f.StringVar(scalingMode, "scaling-mode", "", "Scaling mode (e.g. hpa, manual)")
	f.StringVar(cpuRequest, "cpu-request", "", "CPU request (e.g. 100m)")
	f.StringVar(cpuLimit, "cpu-limit", "", "CPU limit (e.g. 500m)")
	f.StringVar(memoryRequest, "memory-request", "", "Memory request (e.g. 128Mi)")
	f.StringVar(memoryLimit, "memory-limit", "", "Memory limit (e.g. 512Mi)")
	f.StringVar(scalingProfile, "scaling-profile", "", "Scaling profile name")
	f.StringVar(placementPolicy, "placement-policy", "", "Placement policy")
	f.StringVar(canaryInterval, "canary-interval", "", "Canary analysis interval (e.g. 1m)")
	f.BoolVar(isAI, "is-ai", false, "Mark deployment as AI workload")
	f.BoolVar(canaryEnabled, "canary-enabled", false, "Enable canary releases")
	f.IntVar(minReplicas, "min-replicas", 0, "Minimum replica count")
	f.IntVar(maxReplicas, "max-replicas", 0, "Maximum replica count")
	f.IntVar(desiredReplicas, "desired-replicas", 0, "Desired replica count (manual scaling)")
	f.IntVar(cpuTargetUtilization, "cpu-target-utilization", 0, "HPA CPU target utilization percentage")
	f.IntVar(memoryTargetUtilization, "memory-target-utilization", 0, "HPA memory target utilization percentage")
	f.IntVar(canaryStepWeight, "canary-step-weight", 0, "Canary traffic step weight percentage")
	f.IntVar(canaryMaxWeight, "canary-max-weight", 0, "Maximum canary traffic weight percentage")
	f.IntVar(canaryLatencyP99Ms, "canary-latency-p99-ms", 0, "Canary p99 latency threshold in milliseconds")
	f.Float64Var(canarySuccessThreshold, "canary-success-threshold", 0, "Canary success rate threshold (0.0–1.0)")
	f.Float64Var(canaryErrorThreshold, "canary-error-threshold", 0, "Canary error rate threshold (0.0–1.0)")
	f.StringVar(nodeSelector, "node-selector", "", `Node selector as JSON object (e.g. '{"tier":"web"}')`)
	f.StringVar(tolerations, "tolerations", "", `Tolerations as JSON array (e.g. '[{"key":"spot","operator":"Exists","effect":"NoSchedule"}]')`)
	f.StringVar(templateValues, "template-values", "", `Template values as JSON object (e.g. '{"replicas":"3"}')`)
	f.StringVar(templateExtraResources, "template-extra-resources", "", `Extra template resources as JSON object mapping filename to content (e.g. '{"config.yaml":"content"}')`)
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
	var (
		name, namespace, projectID, clusterID, packageName, packageVersion string
		environmentPreset, valuesOverride, scalingMode                     string
		cpuRequest, cpuLimit, memoryRequest, memoryLimit                   string
		scalingProfile, placementPolicy, canaryInterval                    string
		isAI, canaryEnabled                                                bool
		minReplicas, maxReplicas, desiredReplicas                          int
		cpuTargetUtilization, memoryTargetUtilization                      int
		canaryStepWeight, canaryMaxWeight, canaryLatencyP99Ms              int
		canarySuccessThreshold, canaryErrorThreshold                       float64
		nodeSelector, tolerations, templateValues, templateExtraResources  string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new deployment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := deploymentRequestAttrs{
				ProjectID:      &projectID,
				Name:           &name,
				Namespace:      &namespace,
				PackageName:    &packageName,
				PackageVersion: &packageVersion,
			}
			if cmd.Flags().Changed("cluster-id") {
				attrs.ClusterID = &clusterID
			}
			if cmd.Flags().Changed("environment-preset") {
				attrs.EnvironmentPreset = &environmentPreset
			}
			if cmd.Flags().Changed("values-override") {
				attrs.ValuesOverride = &valuesOverride
			}
			if cmd.Flags().Changed("scaling-mode") {
				attrs.ScalingMode = &scalingMode
			}
			if cmd.Flags().Changed("cpu-request") {
				attrs.CPURequest = &cpuRequest
			}
			if cmd.Flags().Changed("cpu-limit") {
				attrs.CPULimit = &cpuLimit
			}
			if cmd.Flags().Changed("memory-request") {
				attrs.MemoryRequest = &memoryRequest
			}
			if cmd.Flags().Changed("memory-limit") {
				attrs.MemoryLimit = &memoryLimit
			}
			if cmd.Flags().Changed("scaling-profile") {
				attrs.ScalingProfile = &scalingProfile
			}
			if cmd.Flags().Changed("placement-policy") {
				attrs.PlacementPolicy = &placementPolicy
			}
			if cmd.Flags().Changed("canary-interval") {
				attrs.CanaryInterval = &canaryInterval
			}
			if cmd.Flags().Changed("is-ai") {
				attrs.IsAI = &isAI
			}
			if cmd.Flags().Changed("canary-enabled") {
				attrs.CanaryEnabled = &canaryEnabled
			}
			if cmd.Flags().Changed("min-replicas") {
				attrs.MinReplicas = &minReplicas
			}
			if cmd.Flags().Changed("max-replicas") {
				attrs.MaxReplicas = &maxReplicas
			}
			if cmd.Flags().Changed("desired-replicas") {
				attrs.DesiredReplicas = &desiredReplicas
			}
			if cmd.Flags().Changed("cpu-target-utilization") {
				attrs.CPUTargetUtilization = &cpuTargetUtilization
			}
			if cmd.Flags().Changed("memory-target-utilization") {
				attrs.MemoryTargetUtilization = &memoryTargetUtilization
			}
			if cmd.Flags().Changed("canary-step-weight") {
				attrs.CanaryStepWeight = &canaryStepWeight
			}
			if cmd.Flags().Changed("canary-max-weight") {
				attrs.CanaryMaxWeight = &canaryMaxWeight
			}
			if cmd.Flags().Changed("canary-latency-p99-ms") {
				attrs.CanaryLatencyP99Ms = &canaryLatencyP99Ms
			}
			if cmd.Flags().Changed("canary-success-threshold") {
				attrs.CanarySuccessThreshold = &canarySuccessThreshold
			}
			if cmd.Flags().Changed("canary-error-threshold") {
				attrs.CanaryErrorThreshold = &canaryErrorThreshold
			}
			if cmd.Flags().Changed("node-selector") {
				parsed, err := parseNodeSelector(nodeSelector)
				if err != nil {
					return err
				}
				attrs.NodeSelector = parsed
			}
			if cmd.Flags().Changed("tolerations") {
				parsed, err := parseTolerations(tolerations)
				if err != nil {
					return err
				}
				attrs.Tolerations = parsed
			}
			if cmd.Flags().Changed("template-values") {
				parsed, err := parseTemplateValues(templateValues)
				if err != nil {
					return err
				}
				attrs.TemplateValues = parsed
			}
			if cmd.Flags().Changed("template-extra-resources") {
				parsed, err := parseTemplateExtraResources(templateExtraResources)
				if err != nil {
					return err
				}
				attrs.TemplateExtraResources = parsed
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
	addDeploymentFlags(cmd,
		&name, &namespace, &projectID, &clusterID, &packageName, &packageVersion,
		&environmentPreset, &valuesOverride, &scalingMode,
		&cpuRequest, &cpuLimit, &memoryRequest, &memoryLimit,
		&scalingProfile, &placementPolicy, &canaryInterval,
		&isAI, &canaryEnabled,
		&minReplicas, &maxReplicas, &desiredReplicas,
		&cpuTargetUtilization, &memoryTargetUtilization,
		&canaryStepWeight, &canaryMaxWeight, &canaryLatencyP99Ms,
		&canarySuccessThreshold, &canaryErrorThreshold,
		&nodeSelector, &tolerations, &templateValues, &templateExtraResources,
	)
	for _, flag := range []string{"name", "namespace", "project-id", "package-name", "package-version"} {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			panic(err)
		}
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var (
		name, namespace, projectID, clusterID, packageName, packageVersion string
		environmentPreset, valuesOverride, scalingMode                     string
		cpuRequest, cpuLimit, memoryRequest, memoryLimit                   string
		scalingProfile, placementPolicy, canaryInterval                    string
		isAI, canaryEnabled                                                bool
		minReplicas, maxReplicas, desiredReplicas                          int
		cpuTargetUtilization, memoryTargetUtilization                      int
		canaryStepWeight, canaryMaxWeight, canaryLatencyP99Ms              int
		canarySuccessThreshold, canaryErrorThreshold                       float64
		nodeSelector, tolerations, templateValues, templateExtraResources  string
	)
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			renderer := ctxutil.RendererFrom(cmd.Context())
			gf := ctxutil.GlobalFlagsFrom(cmd.Context())

			attrs := deploymentRequestAttrs{}
			anyChanged := false

			if cmd.Flags().Changed("name") {
				attrs.Name = &name
				anyChanged = true
			}
			if cmd.Flags().Changed("namespace") {
				attrs.Namespace = &namespace
				anyChanged = true
			}
			if cmd.Flags().Changed("project-id") {
				attrs.ProjectID = &projectID
				anyChanged = true
			}
			if cmd.Flags().Changed("cluster-id") {
				attrs.ClusterID = &clusterID
				anyChanged = true
			}
			if cmd.Flags().Changed("package-name") {
				attrs.PackageName = &packageName
				anyChanged = true
			}
			if cmd.Flags().Changed("package-version") {
				attrs.PackageVersion = &packageVersion
				anyChanged = true
			}
			if cmd.Flags().Changed("environment-preset") {
				attrs.EnvironmentPreset = &environmentPreset
				anyChanged = true
			}
			if cmd.Flags().Changed("values-override") {
				attrs.ValuesOverride = &valuesOverride
				anyChanged = true
			}
			if cmd.Flags().Changed("scaling-mode") {
				attrs.ScalingMode = &scalingMode
				anyChanged = true
			}
			if cmd.Flags().Changed("cpu-request") {
				attrs.CPURequest = &cpuRequest
				anyChanged = true
			}
			if cmd.Flags().Changed("cpu-limit") {
				attrs.CPULimit = &cpuLimit
				anyChanged = true
			}
			if cmd.Flags().Changed("memory-request") {
				attrs.MemoryRequest = &memoryRequest
				anyChanged = true
			}
			if cmd.Flags().Changed("memory-limit") {
				attrs.MemoryLimit = &memoryLimit
				anyChanged = true
			}
			if cmd.Flags().Changed("scaling-profile") {
				attrs.ScalingProfile = &scalingProfile
				anyChanged = true
			}
			if cmd.Flags().Changed("placement-policy") {
				attrs.PlacementPolicy = &placementPolicy
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-interval") {
				attrs.CanaryInterval = &canaryInterval
				anyChanged = true
			}
			if cmd.Flags().Changed("is-ai") {
				attrs.IsAI = &isAI
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-enabled") {
				attrs.CanaryEnabled = &canaryEnabled
				anyChanged = true
			}
			if cmd.Flags().Changed("min-replicas") {
				attrs.MinReplicas = &minReplicas
				anyChanged = true
			}
			if cmd.Flags().Changed("max-replicas") {
				attrs.MaxReplicas = &maxReplicas
				anyChanged = true
			}
			if cmd.Flags().Changed("desired-replicas") {
				attrs.DesiredReplicas = &desiredReplicas
				anyChanged = true
			}
			if cmd.Flags().Changed("cpu-target-utilization") {
				attrs.CPUTargetUtilization = &cpuTargetUtilization
				anyChanged = true
			}
			if cmd.Flags().Changed("memory-target-utilization") {
				attrs.MemoryTargetUtilization = &memoryTargetUtilization
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-step-weight") {
				attrs.CanaryStepWeight = &canaryStepWeight
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-max-weight") {
				attrs.CanaryMaxWeight = &canaryMaxWeight
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-latency-p99-ms") {
				attrs.CanaryLatencyP99Ms = &canaryLatencyP99Ms
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-success-threshold") {
				attrs.CanarySuccessThreshold = &canarySuccessThreshold
				anyChanged = true
			}
			if cmd.Flags().Changed("canary-error-threshold") {
				attrs.CanaryErrorThreshold = &canaryErrorThreshold
				anyChanged = true
			}
			if cmd.Flags().Changed("node-selector") {
				parsed, err := parseNodeSelector(nodeSelector)
				if err != nil {
					return err
				}
				attrs.NodeSelector = parsed
				anyChanged = true
			}
			if cmd.Flags().Changed("tolerations") {
				parsed, err := parseTolerations(tolerations)
				if err != nil {
					return err
				}
				attrs.Tolerations = parsed
				anyChanged = true
			}
			if cmd.Flags().Changed("template-values") {
				parsed, err := parseTemplateValues(templateValues)
				if err != nil {
					return err
				}
				attrs.TemplateValues = parsed
				anyChanged = true
			}
			if cmd.Flags().Changed("template-extra-resources") {
				parsed, err := parseTemplateExtraResources(templateExtraResources)
				if err != nil {
					return err
				}
				attrs.TemplateExtraResources = parsed
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
	addDeploymentFlags(cmd,
		&name, &namespace, &projectID, &clusterID, &packageName, &packageVersion,
		&environmentPreset, &valuesOverride, &scalingMode,
		&cpuRequest, &cpuLimit, &memoryRequest, &memoryLimit,
		&scalingProfile, &placementPolicy, &canaryInterval,
		&isAI, &canaryEnabled,
		&minReplicas, &maxReplicas, &desiredReplicas,
		&cpuTargetUtilization, &memoryTargetUtilization,
		&canaryStepWeight, &canaryMaxWeight, &canaryLatencyP99Ms,
		&canarySuccessThreshold, &canaryErrorThreshold,
		&nodeSelector, &tolerations, &templateValues, &templateExtraResources,
	)
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
