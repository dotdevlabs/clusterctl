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

	"github.com/dotdevlabs/clusterctl/cmd/clusters"
	"github.com/dotdevlabs/clusterctl/cmd/deployments"
	"github.com/dotdevlabs/clusterctl/cmd/packages"
	"github.com/dotdevlabs/clusterctl/cmd/projects"
	"github.com/dotdevlabs/clusterctl/cmd/secrets"
	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

// Execute builds and runs the clusterctl root command.
func Execute() {
	r := root.New(root.BuildConfig{
		Product: "clusterctl",
		Short:   "ClusterControl lifecycle management CLI",
		Version: version.Current("clusterctl"),
		Commands: []*cobra.Command{
			clusters.NewCommand(),
			projects.NewCommand(),
			packages.NewCommand(),
			deployments.NewCommand(),
			secrets.NewCommand(),
		},
		Workflows: aiWorkflows(),
	})

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
			Name:        "Provision a vCluster",
			Description: "Authenticate, create a virtual cluster nested under a parent, and verify its status.",
			Steps: []string{
				"clusterctl auth login",
				"clusterctl clusters create --cluster-type virtual --name my-vcluster --parent-cluster-id <parent-id>",
				"clusterctl clusters get <cluster-id>",
				"clusterctl clusters health-check <cluster-id>",
			},
		},
		{
			Name:        "Create a package then a deployment",
			Description: "Register a Helm chart as a package, then deploy it to a cluster within a project.",
			Steps: []string{
				"clusterctl packages create --name my-chart --source-type helm --source-url https://charts.example.com --source-chart my-chart",
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
	}
}
