package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "deployer",
	Short: "Deploy a containerized service from a GitHub repo to a remote host",
	Long: `deployer ships a service end-to-end: SSH into the host, clone/update the repo,
bring up docker compose, point a Cloudflare DNS record at it, and protect it
with a Cloudflare Zero Trust Access app.

Run ` + "`deployer setup`" + ` once to write ~/.deployer.yml, then ` + "`deployer deploy`" + ` per release.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(deployCmd)
}
