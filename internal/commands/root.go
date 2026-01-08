package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kristofferrisa/powerctl-cli/internal/config"
	"github.com/kristofferrisa/powerctl-cli/internal/output"
)

var (
	cfgFile    string
	formatFlag string
	cfg        *config.Config
	formatter  output.Formatter
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "tibber",
	Short: "Tibber CLI - Power consumption and price data",
	Long: `A command-line interface for Tibber power data.

Get real-time power consumption from your Tibber Pulse,
view electricity prices, and manage your Tibber homes.

Set your API token via TIBBER_TOKEN environment variable
or in ~/.tibber/config.yaml`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}

		// Override format if flag is set
		if formatFlag != "" {
			cfg.Format = formatFlag
		}

		formatter = output.New(cfg.Format)
		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ~/.tibber/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&formatFlag, "format", "f", "", "output format: json, markdown (default: pretty)")
}

// exitWithError prints an error and exits
func exitWithError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
