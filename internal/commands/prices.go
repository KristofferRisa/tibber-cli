package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kristofferrisa/powerctl-cli/internal/api"
)

var pricesCmd = &cobra.Command{
	Use:   "prices",
	Short: "Show electricity prices",
	Long:  `Display current, today's, and tomorrow's electricity prices.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cfg.Validate(); err != nil {
			exitWithError("%v", err)
		}

		client := api.NewClient(cfg.Token)
		ctx := context.Background()

		prices, err := client.GetPrices(ctx, cfg.HomeID)
		if err != nil {
			exitWithError("Failed to fetch prices: %v", err)
		}

		fmt.Println(formatter.FormatPrices(prices, cfg.HomeID))
	},
}

func init() {
	rootCmd.AddCommand(pricesCmd)
}
