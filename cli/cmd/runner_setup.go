package cmd

import (
	"fmt"
	"github.com/semaphoreui/semaphore/services/runners"
	"os"

	"github.com/semaphoreui/semaphore/cli/setup"
	"github.com/semaphoreui/semaphore/util"
	"github.com/spf13/cobra"
)

func init() {
	runnerCmd.AddCommand(runnerSetupCmd)
}

var runnerSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Perform interactive setup",
	Run: func(cmd *cobra.Command, args []string) {
		doRunnerSetup()
	},
}

// nolint: gocyclo
func doRunnerSetup() int {
	config := &util.ConfigType{}

	setup.InteractiveRunnerSetup(config)

	resultConfigPath := setup.SaveConfig(config, "config-runner.json", persistentFlags.configPath)

	util.ConfigInit(resultConfigPath, false)

	if config.Runner.Token != "" {
		if err := os.WriteFile(config.Runner.TokenFile, []byte(config.Runner.Token), 0644); err != nil {
			panic(err)
		}
	}

	if config.Runner.RegistrationToken != "" {
		taskPool := runners.JobPool{}
		err := taskPool.Register()
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf(" Re-launch this program pointing to the configuration file\n\n./semaphore runner --config %v\n\n", resultConfigPath)
	fmt.Printf(" To run as daemon:\n\nnohup ./semaphore runner --config %v &\n\n", resultConfigPath)

	return 0
}
