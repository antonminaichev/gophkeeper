package client

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CLI holds the CLI configuration and state.
type CLI struct {
	// Build info
	Version string
	Commit  string
	Date    string

	// Server connection
	ServerAddr  string
	AccessToken string

	// TLS settings
	TLSEnable             bool
	TLSCAPath             string
	TLSServerName         string
	TLSInsecureSkipVerify bool
}

// NewCLI creates a new CLI instance.
func NewCLI(version, commit, date string) *CLI {
	return &CLI{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}

// CreateCommands creates all CLI commands.
func (cli *CLI) CreateCommands() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gk",
		Short: "GophKeeper CLI",
	}

	// Setup flags
	cli.setupFlags(rootCmd)

	// Add commands
	rootCmd.AddCommand(cli.versionCmd())
	rootCmd.AddCommand(cli.registerCmd())
	rootCmd.AddCommand(cli.loginCmd())
	rootCmd.AddCommand(cli.logoutCmd())
	rootCmd.AddCommand(cli.syncCmd())
	rootCmd.AddCommand(cli.itemCmd())

	return rootCmd
}

// setupFlags configures global flags.
func (cli *CLI) setupFlags(cmd *cobra.Command) {
	// Connection & auth flags
	cmd.PersistentFlags().StringVar(&cli.ServerAddr, "server", "localhost:8090", "gRPC server address")
	cmd.PersistentFlags().StringVar(&cli.AccessToken, "token", os.Getenv("GK_ACCESS_TOKEN"), "Access JWT (optional, overrides saved session)")

	// TLS flags
	cmd.PersistentFlags().BoolVar(&cli.TLSEnable, "tls", false, "Use TLS for gRPC transport")
	cmd.PersistentFlags().StringVar(&cli.TLSCAPath, "tls-ca", "", "Path to custom CA bundle (PEM)")
	cmd.PersistentFlags().StringVar(&cli.TLSServerName, "tls-server-name", "", "Override TLS server name (SNI/verify)")
	cmd.PersistentFlags().BoolVar(&cli.TLSInsecureSkipVerify, "tls-insecure-skip-verify", false, "Skip certificate verification (NOT recommended)")
}

// versionCmd returns the version command.
func (cli *CLI) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gk %s (commit %s, built %s)\n", cli.Version, cli.Commit, cli.Date)
		},
	}
}

// registerCmd returns the register command.
func (cli *CLI) registerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register a new user (interactive)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.register()
		},
	}
}

// loginCmd returns the login command.
func (cli *CLI) loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login (interactive) and save session",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.login()
		},
	}
}

// logoutCmd returns the logout command.
func (cli *CLI) logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear saved session and local cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.logout()
		},
	}
}

// syncCmd returns the sync command.
func (cli *CLI) syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull changes since last sync, update local cache, remember cursor",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.sync()
		},
	}
}

// itemCmd returns the item command group.
func (cli *CLI) itemCmd() *cobra.Command {
	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "Manage vault items",
	}

	// Add subcommands
	itemCmd.AddCommand(cli.itemAddCmd())
	itemCmd.AddCommand(cli.itemListCmd())
	itemCmd.AddCommand(cli.itemGetCmd())
	itemCmd.AddCommand(cli.itemUpdateCmd())
	itemCmd.AddCommand(cli.itemDeleteCmd())

	// Add refresh flag
	var listRefresh bool
	itemCmd.PersistentFlags().BoolVar(&listRefresh, "refresh", false, "Sync before listing")

	return itemCmd
}

// itemAddCmd returns the item add command group.
func (cli *CLI) itemAddCmd() *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add new item: text | login | card | file",
	}

	addCmd.AddCommand(cli.itemAddTextCmd())
	addCmd.AddCommand(cli.itemAddLoginCmd())
	addCmd.AddCommand(cli.itemAddCardCmd())
	addCmd.AddCommand(cli.itemAddFileCmd())

	return addCmd
}

// itemListCmd returns the item list command.
func (cli *CLI) itemListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List items from local cache (use --refresh to sync first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemList()
		},
	}
}

// itemGetCmd returns the item get command.
func (cli *CLI) itemGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get <uuid|human_id|@alias>",
		Short: "Get full item by UUID, human id (#) or alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemGet(args[0])
		},
	}

	// Add flags
	var getReveal bool
	var getSavePath string
	getCmd.Flags().BoolVar(&getReveal, "reveal", false, "Reveal secrets for LOGIN/CARD")
	getCmd.Flags().StringVar(&getSavePath, "save", "", "Save payload to file (TEXT/BINARY)")

	return getCmd
}

// itemUpdateCmd returns the item update command.
func (cli *CLI) itemUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <uuid|human_id|@alias>",
		Short: "Update item (interactive; best for TEXT items)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemUpdate(args[0])
		},
	}
}

// itemDeleteCmd returns the item delete command.
func (cli *CLI) itemDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <uuid|human_id|@alias>",
		Short: "Delete item (soft delete)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemDelete(args[0])
		},
	}
}

// itemAddTextCmd returns the item add text command.
func (cli *CLI) itemAddTextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "text",
		Short: "Create TEXT item (interactive)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemAddText()
		},
	}
}

// itemAddLoginCmd returns the item add login command.
func (cli *CLI) itemAddLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Create LOGIN item (interactive: username/password/url/note + title/alias)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemAddLogin()
		},
	}
}

// itemAddCardCmd returns the item add card command.
func (cli *CLI) itemAddCardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "card",
		Short: "Create CARD item (interactive: holder/PAN/exp/CVC/note + title/alias)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemAddCard()
		},
	}
}

// itemAddFileCmd returns the item add file command.
func (cli *CLI) itemAddFileCmd() *cobra.Command {
	addFileCmd := &cobra.Command{
		Use:   "file",
		Short: "Create BINARY item from file (--path required)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.itemAddFile()
		},
	}

	// Add flags
	var filePathFlag string
	var fileAlias string
	var fileTitle string
	addFileCmd.Flags().StringVar(&filePathFlag, "path", "", "Path to file to upload (required)")
	addFileCmd.Flags().StringVar(&fileAlias, "alias", "", "Optional @alias")
	addFileCmd.Flags().StringVar(&fileTitle, "title", "", "Optional title (meta.title)")
	_ = addFileCmd.MarkFlagRequired("path")

	return addFileCmd
}
