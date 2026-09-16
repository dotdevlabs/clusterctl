package status

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/dotdevlabs/ctlkit/pkg/ctxutil"

	"github.com/dotdevlabs/clusterctl/internal/jsonapi"
)

type statusAttrs struct {
	Version   *string `json:"version"`
	SHA       *string `json:"sha"`
	DBVersion string  `json:"db_version"`
}

// NewCommand returns the "status" cobra.Command.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show ClusterControl API status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := ctxutil.ClientFrom(cmd.Context())
			res, err := jsonapi.GetSingle[statusAttrs](cmd.Context(), client, "/api/v1/status")
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res.Resource.Attributes)
		},
	}
}
