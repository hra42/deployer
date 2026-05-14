package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hra42/deployer/internal/config"
)

// loadConfigFromFlag honors --config when set, otherwise walks the resolution
// order ($DEPLOYER_CONFIG → ~/.deployer.yml → /etc/deployer.yml).
func loadConfigFromFlag(cmd *cobra.Command) (*config.Config, error) {
	var path string
	if cmd.Flags().Lookup("config") != nil {
		path, _ = cmd.Flags().GetString("config")
	}
	if path == "" {
		p, err := config.ResolveReadPath()
		if err != nil {
			return nil, err
		}
		path = p
	}
	return config.Load(path)
}

var rootCmd = &cobra.Command{
	Use:   "deployer",
	Short: "Deploy a containerized service from a GitHub repo to a remote host",
	Long: `deployer ships a service end-to-end: SSH into the host, clone/update the repo,
bring up docker compose, point a Cloudflare DNS record at it, and protect it
with a Cloudflare Zero Trust Access app.

Run ` + "`deployer setup`" + ` once to write ~/.deployer.yml (or ` + "`deployer setup --system`" + `
to write a shared /etc/deployer.yml), then ` + "`deployer deploy`" + ` per release.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(deployCmd)
}
