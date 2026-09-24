package cmd

import (
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/airef"
	"github.com/dotdevlabs/ctlkit/pkg/clierror"
	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"
	"github.com/dotdevlabs/ctlkit/pkg/httpclient"
	"github.com/dotdevlabs/ctlkit/pkg/root"
	"github.com/dotdevlabs/ctlkit/pkg/version"

	"github.com/dotdevlabs/clusterctl/cmd/auth"
	"github.com/dotdevlabs/clusterctl/cmd/clusters"
	"github.com/dotdevlabs/clusterctl/cmd/deployments"
	"github.com/dotdevlabs/clusterctl/cmd/packages"
	"github.com/dotdevlabs/clusterctl/cmd/packageupdatepolicies"
	"github.com/dotdevlabs/clusterctl/cmd/projects"
	"github.com/dotdevlabs/clusterctl/cmd/registrations"
	"github.com/dotdevlabs/clusterctl/cmd/secrets"
	"github.com/dotdevlabs/clusterctl/cmd/status"
	"github.com/dotdevlabs/clusterctl/cmd/templates"
	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

// Execute builds and runs the clusterctl root command.
func Execute() {
	r := root.New(root.BuildConfig{
		Product: "clusterctl",
		Short:   "ClusterControl lifecycle management CLI",
		Version: version.Current("clusterctl"),
		Commands: []*cobra.Command{
			// auth.NewCommand() is intentionally absent: ctlkit's root.New already
			// adds its own "auth" command (providing "auth login"). We extend that
			// command with "auth whoami" below rather than registering a duplicate.
			clusters.NewCommand(),
			deployments.NewCommand(),
			packageupdatepolicies.NewCommand(),
			packages.NewCommand(),
			projects.NewCommand(),
			registrations.NewCommand(),
			secrets.NewCommand(),
			status.NewCommand(),
			templates.NewCommand(),
		},
		Workflows: aiWorkflows(),
	})

	// Extend ctlkit's built-in "auth" command with the ClusterControl-specific
	// "whoami" subcommand (GET /auth). ctlkit registers its "auth" command before
	// product-specific commands, so Find returns it here.
	if authCmd, _, _ := r.Find([]string{"auth"}); authCmd != nil && authCmd != r {
		authCmd.AddCommand(auth.NewWhoamiCmd())
	}

	// Chain our JSON:API transport middleware after ctlkit's PersistentPreRunE.
	// ctlkit creates the client with http.DefaultTransport; we replace it with
	// one that enforces application/vnd.api+json on all requests.
	origPre := r.PersistentPreRunE
	r.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := origPre(cmd, args); err != nil {
			return err
		}
		activeCtx := ctxutil.ActiveContextFrom(cmd.Context())
		if activeCtx == nil {
			return nil
		}
		client := httpclient.NewWithTransport(
			activeCtx.BaseURL,
			activeCtx.Token,
			&jsonapi.Transport{Wrapped: http.DefaultTransport},
		)
		cmd.SetContext(ctxutil.WithClient(cmd.Context(), client))
		return nil
	}

	if err := r.Execute(); err != nil {
		os.Exit(clierror.HandleErr(err, os.Stderr))
	}
}

func aiWorkflows() []airef.Workflow {
	return []airef.Workflow{
		{
			Name:        "Verify identity and service status",
			Description: "Check which organization and token are active, and confirm the API is reachable.",
			Steps: []string{
				"clusterctl auth whoami",
				"clusterctl status",
			},
		},
		{
			Name:        "Provision a vCluster",
			Description: "Authenticate, create a virtual cluster nested under a parent, and verify its status.",
			Steps: []string{
				"clusterctl auth whoami",
				"clusterctl clusters create --cluster-type virtual --name my-vcluster --parent-cluster-id <parent-id>",
				"clusterctl clusters provisioning <cluster-id>",
				"clusterctl clusters get <cluster-id>",
				"clusterctl clusters health-check <cluster-id>",
			},
		},
		{
			Name:        "Create a package then a deployment",
			Description: "Register a Helm chart as a package, then deploy it to a cluster within a project.",
			Steps: []string{
				"clusterctl packages create --name my-chart --source-type helm --source-url https://charts.example.com --source-chart my-chart",
				"clusterctl packages releases list <package-id>",
				"clusterctl deployments create --project-id <project-id> --cluster-id <cluster-id> --name <deployment-name> --namespace default --package-name <package-name> --package-version 1.0.0",
				"clusterctl deployments get <deployment-id>",
			},
		},
		{
			Name:        "Materialize a secret",
			Description: "Create a secret in a project and materialize it to target clusters.",
			Steps: []string{
				"clusterctl secrets create --project-id <project-id> --secret-name app-secrets --key DATABASE_URL --value <secret-value>",
				"clusterctl secrets list --project-id <project-id>",
				"clusterctl secrets materialize --project-id <project-id>",
			},
		},
		{
			Name:        "Check deployment auto-block status and remove a pin",
			Description: "Inspect a deployment for is_auto_blocked/is_pinned status, then remove the pin to allow automatic updates.",
			Steps: []string{
				"clusterctl deployments get <deployment-id>",
				"clusterctl deployments unpin <deployment-id>",
				"clusterctl deployments get <deployment-id>",
			},
		},
		{
			Name:        "Review and manage package update policies",
			Description: "List, create, and update package update policies to control automatic update behavior.",
			Steps: []string{
				"clusterctl package-update-policies list",
				"clusterctl package-update-policies create --deployment-id <deployment-id> --max-attempts 3",
				"clusterctl package-update-policies get <policy-id>",
				"clusterctl package-update-policies update <policy-id> --is-blocked",
			},
		},
		{
			Name:        "Trigger a deployment rollout and inspect update runs",
			Description: "Apply a package update to a deployment and track its update run history.",
			Steps: []string{
				"clusterctl deployments package-update apply <deployment-id>",
				"clusterctl deployments update-runs list <deployment-id>",
				"clusterctl deployments update-runs get <deployment-id> <run-id>",
				"clusterctl deployments rollout <deployment-id>",
			},
		},
		{
			Name:        "Browse available deployment templates",
			Description: "List available deployment templates and inspect a specific template's inputs.",
			Steps: []string{
				"clusterctl templates list",
				"clusterctl templates get <template-id>",
			},
		},
		{
			Name:        "Create and update a deployment template",
			Description: "Create a new deployment template with inputs and manifest files, then update its description or inputs.",
			Steps: []string{
				"clusterctl templates create --slug my-template --name \"My Template\" --inputs '[{\"key\":\"image\",\"type\":\"string\",\"required\":true}]'",
				"clusterctl templates get <template-id>",
				"clusterctl templates update <template-id> --description \"Updated description\"",
			},
		},
	}
}
