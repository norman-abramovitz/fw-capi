package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/cloudfoundry-community/go-uaa"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// flagNameActive is the CLI flag name used to set or query a UAA user's active state.
const flagNameActive = "active"

// createUsersCreateUserCommand creates the create user command.
func createUsersCreateUserCommand() *cobra.Command {
	var (
		email, password, givenName, familyName, phoneNumber, origin string
		active, verified                                            bool
	)

	cmd := &cobra.Command{
		Use:     "create-user <username>",
		Aliases: []string{Create, "add-user", "new-user"},
		Short:   "Create a new user",
		Long: `Create a new user in the UAA database.

The username is required as a positional argument. Other user attributes
can be specified using flags. If not provided via flags, you will be
prompted for required information.`,
		Example: `  # Create user with minimal information
  capi uaa create-user john.doe --email john@example.com

  # Create user with complete profile
  capi uaa create-user jane.smith \
    --email jane@example.com \
    --given-name Jane \
    --family-name Smith \
    --phone-number "+1-555-0123" \
    --verified \
    --active

  # Create user from specific identity provider
  capi uaa create-user ldap.user --email user@company.com --origin ldap`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeCreateUserCommand(args[0], email, password, givenName, familyName, phoneNumber, origin, active, verified)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "User's email address (required)")
	cmd.Flags().StringVar(&password, "password", "", "User's password")
	cmd.Flags().StringVar(&givenName, "given-name", "", "User's given name (first name)")
	cmd.Flags().StringVar(&familyName, "family-name", "", "User's family name (last name)")
	cmd.Flags().StringVar(&phoneNumber, "phone-number", "", "User's phone number")
	cmd.Flags().StringVar(&origin, "origin", "uaa", "Identity provider origin")
	cmd.Flags().BoolVar(&active, flagNameActive, true, "User is active")
	cmd.Flags().BoolVar(&verified, "verified", false, "User is verified")

	return cmd
}

func executeCreateUserCommand(username, email, password, givenName, familyName, phoneNumber, origin string, active, verified bool) error {
	config := loadConfig()

	if GetEffectiveUAAEndpoint(config) == "" {
		err := NewEnhancedError("create user", constants.ErrNoUAAConfigured)
		_ = err.AddSuggestion("Run 'capi uaa target <url>' to set UAA endpoint")

		return err
	}

	// Create UAA client with progress indicator
	var uaaClient *UAAClientWrapper

	err := WrapWithProgress("Connecting to UAA", func() error {
		var err error

		uaaClient, err = NewUAAClient(config)

		return err
	})
	if err != nil {
		return CreateCommonUAAError("create user", err, config.UAAEndpoint)
	}

	if !uaaClient.IsAuthenticated() {
		return CreateCommonUAAError("create user", constants.ErrNotAuthenticated, config.UAAEndpoint)
	}

	// Prompt for required fields if not provided
	email, password, err = promptForUserInput(email, password)
	if err != nil {
		return err
	}

	// Build user object
	user := buildUserFromInput(username, email, password, givenName, familyName, phoneNumber, origin, active, verified)

	// Create user with progress indicator and performance tracking
	var createdUser *uaa.User

	err = WrapWithProgress(fmt.Sprintf("Creating user '%s'", username), func() error {
		return WithPerformanceTracking("create-user", func() error {
			var createErr error

			createdUser, createErr = uaaClient.Client().CreateUser(user)
			if createErr == nil {
				// Invalidate cache for user lookups
				InvalidateUserCache(username)
			}

			if createErr != nil {
				return fmt.Errorf("failed to create user: %w", createErr)
			}

			return nil
		})
	})
	if err != nil {
		enhancedErr := NewEnhancedError("create user", err)
		_ = enhancedErr.AddContext("UAA Endpoint", config.UAAEndpoint)
		_ = enhancedErr.AddContext("Username", username)
		_ = enhancedErr.AddContext("Email", email)
		_ = enhancedErr.AddSuggestion("Verify that the username is unique")
		_ = enhancedErr.AddSuggestion("Check that all required fields are provided")
		_ = enhancedErr.AddSuggestion("Ensure your client has 'scim.write' authority")

		return enhancedErr
	}

	return outputCreatedUser(createdUser)
}

func promptForUserInput(email, password string) (string, string, error) {
	if email == "" {
		log.Print("Email: ")

		_, _ = fmt.Scanln(&email)
	}

	if password == "" {
		log.Print("Password: ")

		passwordBytes, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return "", "", fmt.Errorf("failed to read password: %w", err)
		}

		password = string(passwordBytes)

		_, _ = os.Stdout.WriteString("\n") // Add newline after password input
	}

	return email, password, nil
}

func buildUserFromInput(username, email, password, givenName, familyName, phoneNumber, origin string, active, verified bool) uaa.User {
	user := uaa.User{
		Username: username,
		Password: password,
		Origin:   origin,
		Emails: []uaa.Email{
			{
				Value:   email,
				Primary: &[]bool{true}[0], // Pointer to true
			},
		},
		Active:   &active,
		Verified: &verified,
	}

	// Add name if provided
	if givenName != "" || familyName != "" {
		user.Name = &uaa.UserName{
			GivenName:  givenName,
			FamilyName: familyName,
		}
	}

	// Add phone number if provided
	if phoneNumber != "" {
		user.PhoneNumbers = []uaa.PhoneNumber{
			{
				Value: phoneNumber,
			},
		}
	}

	return user
}

func outputCreatedUser(createdUser *uaa.User) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(createdUser) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; this is CLI user-creation output, not persisted config
		if err != nil {
			return fmt.Errorf("encoding created user to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(createdUser) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; this is CLI user-creation output, not persisted config
		if err != nil {
			return fmt.Errorf("encoding created user to YAML: %w", err)
		}

		return nil
	default:
		return displayUserTable(createdUser)
	}
}

// createUsersGetUserCommand creates the get user command.
func createUsersGetUserCommand() *cobra.Command {
	var attributes string

	cmd := &cobra.Command{
		Use:     "get-user <username>",
		Aliases: []string{"get", "show-user", UserCommandName},
		Short:   "Get user details",
		Long: `Look up a user by username and display detailed information.

The command will search for the user by username and display all available
user attributes including groups, metadata, and authentication information.`,
		Example: `  # Get complete user information
  capi uaa get-user john.doe

  # Get specific user attributes only
  capi uaa get-user john.doe --attributes userName,email,active

  # Get user in JSON format for scripting
  capi uaa get-user john.doe --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()
			username := args[0]

			if GetEffectiveUAAEndpoint(config) == "" {
				return constants.ErrNoUAAConfigured
			}

			// Create UAA client
			uaaClient, err := NewUAAClient(config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			if !uaaClient.IsAuthenticated() {
				return constants.ErrNotAuthenticated
			}

			// Get user by username with caching
			var user *uaa.User

			err = WithPerformanceTracking("get-user", func() error {
				var getUserErr error

				user, getUserErr = CachedUserLookup(uaaClient, username)

				return getUserErr
			})
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			// Display user
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(user) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; CLI get-user output mirrors UAA API response
				if err != nil {
					return fmt.Errorf("encoding user to JSON: %w", err)
				}

				return nil
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(user) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; CLI get-user output mirrors UAA API response
				if err != nil {
					return fmt.Errorf("encoding user to YAML: %w", err)
				}

				return nil
			default:
				return displayUserTable(user)
			}
		},
	}

	cmd.Flags().StringVar(&attributes, "attributes", "", "Comma-separated list of attributes to return")

	return cmd
}

// createUsersListUsersCommand creates the list users command.
func createUsersListUsersCommand() *cobra.Command {
	var (
		filter, sortBy, attributes string
		sortOrder                  string
		count, startIndex          int
		all                        bool
	)

	cmd := &cobra.Command{
		Use:     "list-users",
		Aliases: []string{List, "users", "ls"},
		Short:   "List users",
		Long: `Search and list users with optional SCIM filters.

SCIM filters allow complex queries on user attributes. Examples:
- userName eq "john@example.com"
- active eq true
- origin eq "ldap"
- meta.created gt "2023-01-01T00:00:00.000Z"`,
		Example: `  # List all active users
  capi uaa list-users --filter 'active eq true'

  # List users from specific domain
  capi uaa list-users --filter 'email co "example.com"'

  # List recent users with specific attributes
  capi uaa list-users \
    --filter 'meta.created gt "2023-01-01T00:00:00.000Z"' \
    --attributes userName,email,meta.created \
    --sort-by meta.created \
    --sort-order desc

  # Get all users (auto-pagination)
  capi uaa list-users --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeListUsersCommand(filter, sortBy, attributes, sortOrder, count, startIndex, all)
		},
	}

	cmd.Flags().StringVar(&filter, "filter", "", "SCIM filter expression")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Attribute to sort by")
	cmd.Flags().StringVar(&sortOrder, "sort-order", "ascending", "Sort order (ascending, descending)")
	cmd.Flags().StringVar(&attributes, "attributes", "", "Comma-separated list of attributes to return")
	cmd.Flags().IntVar(&count, "count", constants.StandardPageSize, "Number of results per page")
	cmd.Flags().IntVar(&startIndex, "start-index", 1, "Starting index for pagination")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all users across all pages")

	return cmd
}

func executeListUsersCommand(filter, sortBy, attributes, sortOrder string, count, startIndex int, all bool) error {
	config := loadConfig()

	if GetEffectiveUAAEndpoint(config) == "" {
		return constants.ErrNoUAAConfigured
	}

	uaaClient, err := NewUAAClient(config)
	if err != nil {
		return fmt.Errorf("failed to create UAA client: %w", err)
	}

	if !uaaClient.IsAuthenticated() {
		return constants.ErrNotAuthenticated
	}

	uaaSortOrder := parseSortOrder(sortOrder)

	users, err := fetchUsers(uaaClient, filter, sortBy, attributes, uaaSortOrder, count, startIndex, all)
	if err != nil {
		return err
	}

	return outputUsersList(users)
}

func parseSortOrder(sortOrder string) uaa.SortOrder {
	switch strings.ToLower(sortOrder) {
	case Descending, "desc":
		return Descending
	default:
		return uaa.SortAscending
	}
}

func fetchUsers(uaaClient *UAAClientWrapper, filter, sortBy, attributes string, uaaSortOrder uaa.SortOrder, count, startIndex int, all bool) ([]uaa.User, error) {
	var users []uaa.User

	err := WithPerformanceTracking("list-users", func() error {
		var listErr error

		if all {
			pagination := NewOptimizedPagination(uaaClient)
			users, listErr = pagination.GetAllUsers(filter, sortBy, attributes, uaaSortOrder)
		} else {
			users, _, listErr = uaaClient.Client().ListUsers(filter, sortBy, attributes, uaaSortOrder, startIndex, count)
		}

		if listErr != nil {
			return fmt.Errorf("failed to list users: %w", listErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

func outputUsersList(users []uaa.User) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(users) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; CLI list-users output mirrors UAA API response
		if err != nil {
			return fmt.Errorf("encoding users to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(users) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; CLI list-users output mirrors UAA API response
		if err != nil {
			return fmt.Errorf("encoding users to YAML: %w", err)
		}

		return nil
	default:
		return displayUsersTable(users)
	}
}

// createUsersUpdateUserCommand creates the update user command.
func createUsersUpdateUserCommand() *cobra.Command {
	var (
		email, givenName, familyName, phoneNumber string
		activeStr, verifiedStr                    string
		active, verified                          *bool
	)

	cmd := &cobra.Command{
		Use:   "update-user <username>",
		Short: "Update user attributes",
		Long: `Update attributes for an existing user.

Only the specified attributes will be updated. Unspecified attributes
will remain unchanged.`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return parseUpdateUserBoolFlags(activeStr, verifiedStr, &active, &verified)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()
			username := args[0]

			uaaClient, err := getAuthenticatedUAAClient(config)
			if err != nil {
				return err
			}

			existingUser, err := uaaClient.Client().GetUserByUsername(username, "", "")
			if err != nil {
				return fmt.Errorf("failed to get existing user: %w", err)
			}

			applyUserUpdates(existingUser, email, givenName, familyName, phoneNumber, active, verified)

			updatedUser, err := uaaClient.Client().UpdateUser(*existingUser)
			if err != nil {
				return fmt.Errorf("failed to update user: %w", err)
			}

			return outputUser(updatedUser)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "User's email address")
	cmd.Flags().StringVar(&givenName, "given-name", "", "User's given name (first name)")
	cmd.Flags().StringVar(&familyName, "family-name", "", "User's family name (last name)")
	cmd.Flags().StringVar(&phoneNumber, "phone-number", "", "User's phone number")
	cmd.Flags().StringVar(&activeStr, flagNameActive, "", "User is active (true/false)")
	cmd.Flags().StringVar(&verifiedStr, "verified", "", "User is verified (true/false)")

	return cmd
}

// parseUpdateUserBoolFlags parses the active and verified string flags into boolean pointers.
func parseUpdateUserBoolFlags(activeStr, verifiedStr string, active, verified **bool) error {
	if activeStr != "" {
		val, err := strconv.ParseBool(activeStr)
		if err != nil {
			return fmt.Errorf("%w: %s", constants.ErrInvalidActive, activeStr)
		}

		*active = &val
	}

	if verifiedStr != "" {
		val, err := strconv.ParseBool(verifiedStr)
		if err != nil {
			return fmt.Errorf("%w: %s", constants.ErrInvalidVerified, verifiedStr)
		}

		*verified = &val
	}

	return nil
}

// applyUserUpdates applies the specified updates to the existing user object.
func applyUserUpdates(user *uaa.User, email, givenName, familyName, phoneNumber string, active, verified *bool) {
	if email != "" {
		user.Emails = []uaa.Email{
			{
				Value:   email,
				Primary: &[]bool{true}[0],
			},
		}
	}

	if givenName != "" || familyName != "" {
		if user.Name == nil {
			user.Name = &uaa.UserName{}
		}

		if givenName != "" {
			user.Name.GivenName = givenName
		}

		if familyName != "" {
			user.Name.FamilyName = familyName
		}
	}

	if phoneNumber != "" {
		user.PhoneNumbers = []uaa.PhoneNumber{
			{
				Value: phoneNumber,
			},
		}
	}

	if active != nil {
		user.Active = active
	}

	if verified != nil {
		user.Verified = verified
	}
}

// getAuthenticatedUAAClient creates and validates a UAA client.
func getAuthenticatedUAAClient(config *Config) (*UAAClientWrapper, error) {
	if GetEffectiveUAAEndpoint(config) == "" {
		return nil, constants.ErrNoUAAConfigured
	}

	uaaClient, err := NewUAAClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create UAA client: %w", err)
	}

	if !uaaClient.IsAuthenticated() {
		return nil, constants.ErrNotAuthenticated
	}

	return uaaClient, nil
}

// outputUser formats and displays user information based on the output format.
func outputUser(user *uaa.User) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(user) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; CLI update-user output mirrors UAA API response
		if err != nil {
			return fmt.Errorf("encoding updated user to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(user) //#nosec G117 -- nolint:gosec -- password field intentionally marshaled; CLI update-user output mirrors UAA API response
		if err != nil {
			return fmt.Errorf("encoding updated user to YAML: %w", err)
		}

		return nil
	default:
		return displayUserTable(user)
	}
}

// createUsersActivateUserCommand creates the activate user command.
func createUsersActivateUserCommand() *cobra.Command {
	return createUserActivationCommand(
		"activate-user <username>",
		"Activate a user account",
		"Activate a user account by username",
		true,
		"User '%s' has been activated\n",
	)
}

// createUsersDeactivateUserCommand creates the deactivate user command.
func createUsersDeactivateUserCommand() *cobra.Command {
	return createUserActivationCommand(
		"deactivate-user <username>",
		"Deactivate a user account",
		"Deactivate a user account by username",
		false,
		"User '%s' has been deactivated\n",
	)
}

// createUsersDeleteUserCommand creates the delete user command.
func createUsersDeleteUserCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete-user <username>",
		Aliases: []string{"delete", "remove-user", "rm"},
		Short:   "Delete a user",
		Long:    "Delete a user by username. This action is irreversible.",
		Example: `  # Delete user with confirmation prompt
  capi uaa delete-user john.doe

  # Force delete without confirmation
  capi uaa delete-user john.doe --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()
			username := args[0]

			if GetEffectiveUAAEndpoint(config) == "" {
				return constants.ErrNoUAAConfigured
			}

			// Create UAA client
			uaaClient, err := NewUAAClient(config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			if !uaaClient.IsAuthenticated() {
				return constants.ErrNotAuthenticated
			}

			// Get user to get ID and display info
			user, err := uaaClient.Client().GetUserByUsername(username, "", "")
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			// Confirm deletion unless --force is used
			if !ConfirmAction(fmt.Sprintf("Are you sure you want to delete user '%s' (ID: %s)?", username, user.ID), force) {
				_, _ = os.Stdout.WriteString("User deletion cancelled\n")

				return nil
			}

			// Delete user
			_, err = uaaClient.Client().DeleteUser(user.ID)
			if err != nil {
				return fmt.Errorf("failed to delete user: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "User '%s' has been deleted\n", username)

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force deletion without confirmation")

	return cmd
}

// Helper functions for user display

func displayUserTable(user *uaa.User) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	if user.ID != "" {
		_ = table.Append("ID", user.ID)
	}

	if user.Username != "" {
		_ = table.Append("Username", user.Username)
	}

	if user.Origin != "" {
		_ = table.Append("Origin", user.Origin)
	}

	// Display name
	if user.Name != nil {
		if user.Name.GivenName != "" {
			_ = table.Append("Given Name", user.Name.GivenName)
		}

		if user.Name.FamilyName != "" {
			_ = table.Append("Family Name", user.Name.FamilyName)
		}
	}

	// Display primary email
	if len(user.Emails) > 0 {
		_ = table.Append("Email", user.Emails[0].Value)
	}

	// Display phone numbers
	if len(user.PhoneNumbers) > 0 {
		_ = table.Append("Phone", user.PhoneNumbers[0].Value)
	}

	// Display status
	if user.Active != nil {
		_ = table.Append("Active", strconv.FormatBool(*user.Active))
	}

	if user.Verified != nil {
		_ = table.Append("Verified", strconv.FormatBool(*user.Verified))
	}

	// Display metadata
	if user.Meta != nil {
		if user.Meta.Created != "" {
			_ = table.Append("Created", user.Meta.Created)
		}

		if user.Meta.LastModified != "" {
			_ = table.Append("Last Modified", user.Meta.LastModified)
		}

		if user.Meta.Version > 0 {
			_ = table.Append("Version", strconv.Itoa(user.Meta.Version))
		}
	}

	// Display groups count
	if len(user.Groups) > 0 {
		_ = table.Append("Groups", fmt.Sprintf("%d groups", len(user.Groups)))
	}

	_ = table.Render()

	return nil
}

func displayUsersTable(users []uaa.User) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Username", "Email", "Given Name", "Active", "Verified", "Origin", "Created")

	for _, user := range users {
		username := user.Username

		email := ""
		if len(user.Emails) > 0 {
			email = user.Emails[0].Value
		}

		givenName := ""
		if user.Name != nil {
			givenName = user.Name.GivenName
		}

		active := "unknown"
		if user.Active != nil {
			active = strconv.FormatBool(*user.Active)
		}

		verified := "unknown"
		if user.Verified != nil {
			verified = strconv.FormatBool(*user.Verified)
		}

		origin := user.Origin

		created := ""
		if user.Meta != nil && user.Meta.Created != "" {
			created = user.Meta.Created
		}

		_ = table.Append(username, email, givenName, active, verified, origin, created)
	}

	_ = table.Render()

	return nil
}
