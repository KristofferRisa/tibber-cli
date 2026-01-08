package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kristofferrisa/powerctl-cli/internal/api"
	"github.com/kristofferrisa/powerctl-cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Create, view, and update your Tibber CLI configuration.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long: `Create a new configuration file interactively.

This will guide you through setting up your Tibber API token
and other settings. Your token can be found at:
https://developer.tibber.com/settings/access-token`,
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("Tibber CLI Configuration Setup")
		fmt.Println("===============================")
		fmt.Println()
		fmt.Println("Get your API token from: https://developer.tibber.com/settings/access-token")
		fmt.Println()

		// Get token
		fmt.Print("Enter your Tibber API token: ")
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)

		if token == "" {
			exitWithError("Token is required")
		}

		// Validate token by fetching homes
		fmt.Println("\nValidating token...")
		client := api.NewClient(token)
		homes, err := client.GetHomes(cmd.Context())
		if err != nil {
			exitWithError("Invalid token: %v", err)
		}

		fmt.Printf("Found %d home(s)\n\n", len(homes))

		// Let user select default home if multiple
		var homeID string
		if len(homes) > 1 {
			fmt.Println("Select default home:")
			for i, home := range homes {
				name := home.AppNickname
				if name == "" {
					name = home.Address.Address1
				}
				pulse := ""
				if home.Features.RealTimeConsumptionEnabled {
					pulse = " [Pulse]"
				}
				fmt.Printf("  %d) %s%s\n", i+1, name, pulse)
			}
			fmt.Print("\nEnter number (or press Enter to skip): ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice != "" {
				var idx int
				fmt.Sscanf(choice, "%d", &idx)
				if idx > 0 && idx <= len(homes) {
					homeID = homes[idx-1].ID
				}
			}
		} else if len(homes) == 1 {
			homeID = homes[0].ID
		}

		// Get format preference
		fmt.Print("Default output format (pretty/json/markdown) [pretty]: ")
		format, _ := reader.ReadString('\n')
		format = strings.TrimSpace(format)
		if format == "" {
			format = "pretty"
		}

		// Create config
		configData := map[string]string{
			"token":  token,
			"format": format,
		}
		if homeID != "" {
			configData["home_id"] = homeID
		}

		// Ensure config directory exists
		if err := config.EnsureConfigDir(); err != nil {
			exitWithError("Failed to create config directory: %v", err)
		}

		// Write config file
		configPath := config.DefaultConfigPath()
		yamlData, err := yaml.Marshal(configData)
		if err != nil {
			exitWithError("Failed to create config: %v", err)
		}

		if err := os.WriteFile(configPath, yamlData, 0600); err != nil {
			exitWithError("Failed to write config file: %v", err)
		}

		fmt.Printf("\nConfiguration saved to %s\n", configPath)
		fmt.Println("\nYou can now use the CLI:")
		fmt.Println("  tibber home     - View your home info")
		fmt.Println("  tibber prices   - View electricity prices")
		fmt.Println("  tibber live     - Stream live power data")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath := config.DefaultConfigPath()

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Println("No configuration file found.")
			fmt.Println()
			fmt.Println("Run 'tibber config init' to create one, or set TIBBER_TOKEN environment variable.")
			return
		}

		// Read and display config
		data, err := os.ReadFile(configPath)
		if err != nil {
			exitWithError("Failed to read config: %v", err)
		}

		var configData map[string]string
		if err := yaml.Unmarshal(data, &configData); err != nil {
			exitWithError("Failed to parse config: %v", err)
		}

		fmt.Printf("Configuration file: %s\n\n", configPath)

		// Mask token for security
		if token, ok := configData["token"]; ok && len(token) > 8 {
			configData["token"] = token[:4] + "..." + token[len(token)-4:]
		}

		for key, value := range configData {
			fmt.Printf("  %s: %s\n", key, value)
		}

		// Show environment overrides
		if envToken := os.Getenv("TIBBER_TOKEN"); envToken != "" {
			fmt.Println("\nEnvironment overrides:")
			fmt.Printf("  TIBBER_TOKEN: %s...%s\n", envToken[:4], envToken[len(envToken)-4:])
		}
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Long:  `Display the path to the configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.DefaultConfigPath())
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a specific configuration value.

Available keys:
  token    - Your Tibber API token
  home_id  - Default home ID
  format   - Output format (pretty, json, or markdown)`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		// Validate key
		validKeys := map[string]bool{"token": true, "home_id": true, "format": true}
		if !validKeys[key] {
			exitWithError("Invalid key: %s. Valid keys: token, home_id, format", key)
		}

		// Validate format value
		if key == "format" && value != "pretty" && value != "markdown" && value != "json" {
			exitWithError("Invalid format: %s. Use 'pretty', 'json', or 'markdown'", value)
		}

		// Ensure config directory exists
		if err := config.EnsureConfigDir(); err != nil {
			exitWithError("Failed to create config directory: %v", err)
		}

		configPath := config.DefaultConfigPath()

		// Read existing config or create new
		configData := make(map[string]string)
		if data, err := os.ReadFile(configPath); err == nil {
			yaml.Unmarshal(data, &configData)
		}

		// Update value
		configData[key] = value

		// Write config
		yamlData, err := yaml.Marshal(configData)
		if err != nil {
			exitWithError("Failed to create config: %v", err)
		}

		if err := os.WriteFile(configPath, yamlData, 0600); err != nil {
			exitWithError("Failed to write config file: %v", err)
		}

		fmt.Printf("Set %s in %s\n", key, configPath)
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration file in editor",
	Long:  `Open the configuration file in your default editor.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath := config.DefaultConfigPath()

		// Ensure config exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Create empty config
			if err := config.EnsureConfigDir(); err != nil {
				exitWithError("Failed to create config directory: %v", err)
			}

			template := `# Tibber CLI Configuration
# Get your token from: https://developer.tibber.com/settings/access-token

token: ""
# home_id: ""
# format: pretty  # Options: pretty, json, markdown
`
			if err := os.WriteFile(configPath, []byte(template), 0600); err != nil {
				exitWithError("Failed to create config file: %v", err)
			}
		}

		// Get editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			// Try common editors
			for _, e := range []string{"vim", "nano", "vi"} {
				if _, err := os.Stat(filepath.Join("/usr/bin", e)); err == nil {
					editor = e
					break
				}
			}
		}

		if editor == "" {
			fmt.Printf("Config file location: %s\n", configPath)
			fmt.Println("No editor found. Set EDITOR environment variable or edit the file manually.")
			return
		}

		fmt.Printf("Opening %s with %s\n", configPath, editor)
		fmt.Println("(editor will open in your terminal)")
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configEditCmd)
	rootCmd.AddCommand(configCmd)
}
