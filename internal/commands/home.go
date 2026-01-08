package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kristofferrisa/powerctl-cli/internal/api"
)

var homeCmd = &cobra.Command{
	Use:   "home",
	Short: "Show home information",
	Long:  `Display information about your Tibber homes including address, size, and Pulse status.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cfg.Validate(); err != nil {
			exitWithError("%v", err)
		}

		client := api.NewClient(cfg.Token)
		ctx := context.Background()

		homes, err := client.GetHomes(ctx)
		if err != nil {
			exitWithError("Failed to fetch homes: %v", err)
		}

		if len(homes) == 0 {
			exitWithError("No homes found")
		}

		// If home_id is configured, show only that home
		if cfg.HomeID != "" {
			for _, home := range homes {
				if home.ID == cfg.HomeID {
					fmt.Println(formatter.FormatHome(&home))
					return
				}
			}
			exitWithError("Home with ID %s not found", cfg.HomeID)
		}

		// Show all homes
		fmt.Println(formatter.FormatHomes(homes))
	},
}

func init() {
	rootCmd.AddCommand(homeCmd)
}
