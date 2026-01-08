package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kristofferrisa/powerctl-cli/internal/api"
	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

var (
	liveHomeID string
)

var liveCmd = &cobra.Command{
	Use:   "live",
	Short: "Stream real-time power consumption",
	Long: `Stream live power consumption data from your Tibber Pulse.

Requires a Tibber Pulse device connected to your home.
Press Ctrl+C to stop the stream.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cfg.Validate(); err != nil {
			exitWithError("%v", err)
		}

		homeID := liveHomeID
		if homeID == "" {
			homeID = cfg.HomeID
		}

		// If no home ID, fetch homes and use first one with Pulse
		if homeID == "" {
			client := api.NewClient(cfg.Token)
			homes, err := client.GetHomes(context.Background())
			if err != nil {
				exitWithError("Failed to fetch homes: %v", err)
			}

			for _, home := range homes {
				if home.Features.RealTimeConsumptionEnabled {
					homeID = home.ID
					break
				}
			}

			if homeID == "" {
				exitWithError("No home with Pulse found. Ensure your Tibber Pulse is connected.")
			}
		}

		// Set up signal handling for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		liveClient := api.NewLiveClient(cfg.Token, homeID)

		fmt.Fprintf(os.Stderr, "Connecting to live stream...\n")

		err := liveClient.Subscribe(ctx, func(m *models.LiveMeasurement) error {
			// Clear screen for markdown, just print for JSON
			if cfg.Format == "json" {
				fmt.Println(formatter.FormatLiveMeasurement(m))
			} else {
				// ANSI escape to clear screen and move cursor to top
				fmt.Print("\033[2J\033[H")
				fmt.Println(formatter.FormatLiveMeasurement(m))
			}
			return nil
		})

		if err != nil && ctx.Err() == nil {
			exitWithError("Stream error: %v", err)
		}
	},
}

func init() {
	liveCmd.Flags().StringVar(&liveHomeID, "home-id", "", "specific home ID to monitor")
	rootCmd.AddCommand(liveCmd)
}
