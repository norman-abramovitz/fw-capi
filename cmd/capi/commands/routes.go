package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewRoutesCommand creates the routes command group.
func NewRoutesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "routes",
		Aliases: []string{"route"},
		Short:   "Manage routes",
		Long:    "List and manage Cloud Foundry routes",
	}

	cmd.AddCommand(newRoutesListCommand())
	cmd.AddCommand(newRoutesCreateCommand())
	cmd.AddCommand(newRoutesDeleteCommand())
	cmd.AddCommand(newRoutesShareCommand())
	cmd.AddCommand(newRoutesUnshareCommand())
	cmd.AddCommand(newRoutesTransferCommand())
	cmd.AddCommand(newRoutesListSharedCommand())

	return cmd
}

func newRoutesListCommand() *cobra.Command {
	var (
		spaceName  string
		domainName string
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List routes",
		Long:  "List all routes the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoutesList(cmd, spaceName, domainName)
		},
	}

	cmd.Flags().StringVarP(&spaceName, spaceKey, "s", "", "filter by space name")
	cmd.Flags().StringVarP(&domainName, "domain", "d", "", "filter by domain name")

	return cmd
}

func runRoutesList(cmd *cobra.Command, spaceName, domainName string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	params := capi.NewQueryParams()

	// Apply space filter
	err = applySpaceFilter(ctx, client, params, spaceName)
	if err != nil {
		return err
	}

	// Apply domain filter
	err = applyDomainFilter(ctx, client, params, domainName)
	if err != nil {
		return err
	}

	routes, err := client.Routes().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	// Output results
	return renderRoutesOutput(routes.Resources)
}

func applySpaceFilter(ctx context.Context, client capi.Client, params *capi.QueryParams, spaceName string) error {
	if spaceName != "" {
		spaceGUID, err := resolveSpaceGUIDWithOrgFilter(ctx, client, spaceName)
		if err != nil {
			return err
		}

		params.WithFilter("space_guids", spaceGUID)
	} else if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		// Use targeted space
		params.WithFilter("space_guids", spaceGUID)
	}

	return nil
}

func applyDomainFilter(ctx context.Context, client capi.Client, params *capi.QueryParams, domainName string) error {
	if domainName == "" {
		return nil
	}

	domainGUID, err := resolveDomainGUID(ctx, client, domainName)
	if err != nil {
		return err
	}

	params.WithFilter("domain_guids", domainGUID)

	return nil
}

func resolveDomainGUID(ctx context.Context, client capi.Client, domainName string) (string, error) {
	domainParams := capi.NewQueryParams()
	domainParams.WithFilter("names", domainName)

	domains, err := client.Domains().List(ctx, domainParams)
	if err != nil {
		return "", fmt.Errorf("failed to find domain: %w", err)
	}

	if len(domains.Resources) == 0 {
		return "", fmt.Errorf("domain '%s': %w", domainName, ErrDomainNotFound)
	}

	return domains.Resources[0].GUID, nil
}

func renderRoutesOutput(routes []capi.Route) error {
	renderer := &StandardOutputRenderer[capi.Route]{
		RenderTable: func(resources []capi.Route, pag *capi.Pagination, allPgs bool) error {
			return renderRoutesTable(resources)
		},
	}

	output := viper.GetString("output")

	return renderer.Render(routes, nil, false, output)
}

func renderRoutesTable(routes []capi.Route) error {
	if len(routes) == 0 {
		_, _ = os.Stdout.WriteString("No routes found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("URL", "GUID", "Protocol", "Host", "Path", "Port", "Destinations", "Created", "Updated")

	for _, route := range routes {
		path := formatRoutePath(route.Path)
		port := formatRoutePort(route.Port)
		destinations := formatRouteDestinations(route.Destinations)
		created := formatRouteTimestamp(route.CreatedAt)
		updated := formatRouteTimestamp(route.UpdatedAt)

		_ = table.Append(route.URL, route.GUID, route.Protocol, route.Host, path, port, destinations, created, updated)
	}

	_ = table.Render()

	return nil
}

func formatRoutePath(path string) string {
	if path == "" {
		return "/"
	}

	return path
}

func formatRoutePort(port *int) string {
	if port != nil {
		return strconv.Itoa(*port)
	}

	return ""
}

func formatRouteDestinations(destinations []capi.RouteDestination) string {
	destCount := len(destinations)
	if destCount == 0 {
		return ""
	}

	if destCount == 1 {
		return "1 destination"
	}

	return fmt.Sprintf("%d destinations", destCount)
}

func formatRouteTimestamp(timestamp time.Time) string {
	if timestamp.IsZero() {
		return ""
	}

	return timestamp.Format(TimeFormatDisplay)
}

func newRoutesCreateCommand() *cobra.Command {
	var (
		spaceName string
		hostname  string
		path      string
		port      int
	)

	cmd := &cobra.Command{
		Use:   "create DOMAIN_NAME",
		Short: "Create a route",
		Long:  "Create a new Cloud Foundry route in a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domainName := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			domain, err := findDomainByName(ctx, client, domainName)
			if err != nil {
				return err
			}

			spaceGUID, err := resolveSpaceGUID(ctx, client, spaceName)
			if err != nil {
				return err
			}

			createReq := buildRouteCreateRequest(spaceGUID, domain.GUID, hostname, path, port, cmd.Flags().Changed("port"))

			route, err := client.Routes().Create(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create route: %w", err)
			}

			printCreatedRoute(route)

			return nil
		},
	}

	cmd.Flags().StringVarP(&spaceName, spaceKey, "s", "", "space name (defaults to targeted space)")
	cmd.Flags().StringVar(&hostname, "hostname", "", "hostname for the route")
	cmd.Flags().StringVar(&path, "path", "", "path for the route")
	cmd.Flags().IntVar(&port, "port", 0, "port for the route (for TCP routes)")

	return cmd
}

func findDomainByName(ctx context.Context, client capi.Client, domainName string) (*capi.Domain, error) {
	domainParams := capi.NewQueryParams()
	domainParams.WithFilter("names", domainName)

	domains, err := client.Domains().List(ctx, domainParams)
	if err != nil {
		return nil, fmt.Errorf("failed to find domain: %w", err)
	}

	if len(domains.Resources) == 0 {
		return nil, fmt.Errorf("domain '%s': %w", domainName, ErrDomainNotFound)
	}

	return &domains.Resources[0], nil
}

func buildRouteCreateRequest(spaceGUID, domainGUID, hostname, path string, port int, portChanged bool) *capi.RouteCreateRequest {
	createReq := &capi.RouteCreateRequest{
		Relationships: capi.RouteRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{GUID: spaceGUID},
			},
			Domain: capi.Relationship{
				Data: &capi.RelationshipData{GUID: domainGUID},
			},
		},
	}

	if hostname != "" {
		createReq.Host = &hostname
	}

	if path != "" {
		createReq.Path = &path
	}

	if portChanged {
		createReq.Port = &port
	}

	return createReq
}

func printCreatedRoute(route *capi.Route) {
	_, _ = fmt.Fprintf(os.Stdout, "Successfully created route: %s\n", route.URL)
	_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", route.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Protocol: %s\n", route.Protocol)
	_, _ = fmt.Fprintf(os.Stdout, "  Host: %s\n", route.Host)

	if route.Path != "" {
		_, _ = fmt.Fprintf(os.Stdout, "  Path: %s\n", route.Path)
	}

	if route.Port != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Port: %d\n", *route.Port)
	}
}

func newRoutesDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete ROUTE_GUID_OR_URL",
		Short: "Delete a route",
		Long:  "Delete a Cloud Foundry route by GUID or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			routeIdentifier := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			routeGUID, routeURL, err := findRouteByIdentifier(ctx, client, routeIdentifier)
			if err != nil {
				return err
			}

			if !force {
				confirmed := confirmRouteDeletion(routeURL)
				if !confirmed {
					_, _ = os.Stdout.WriteString("Deletion cancelled.\n")

					return nil
				}
			}

			return deleteRoute(ctx, client, routeGUID, routeURL)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force deletion without confirmation")

	return cmd
}

func findRouteByIdentifier(ctx context.Context, client capi.Client, routeIdentifier string) (string, string, error) {
	// Try as GUID first
	route, err := client.Routes().Get(ctx, routeIdentifier)
	if err == nil {
		return route.GUID, route.URL, nil
	}

	// Try to find by URL
	params := capi.NewQueryParams()

	routes, err := client.Routes().List(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("failed to list routes: %w", err)
	}

	for _, r := range routes.Resources {
		if r.URL == routeIdentifier {
			return r.GUID, r.URL, nil
		}
	}

	return "", "", fmt.Errorf("route '%s': %w", routeIdentifier, ErrRouteNotFound)
}

func confirmRouteDeletion(routeURL string) bool {
	_, _ = fmt.Fprintf(os.Stdout, "Are you sure you want to delete route '%s'? (y/N): ", routeURL)

	var response string

	_, _ = fmt.Scanln(&response)

	return response == "y" || response == "Y" || response == constants.ConfirmationYes || response == "YES"
}

func deleteRoute(ctx context.Context, client capi.Client, routeGUID, routeURL string) error {
	job, err := client.Routes().Delete(ctx, routeGUID)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	if job != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Route deletion job created: %s\n", job.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "Successfully initiated deletion of route '%s'\n", routeURL)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Successfully deleted route '%s'\n", routeURL)
	}

	return nil
}

func newRoutesShareCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "share ROUTE_GUID_OR_URL SPACE_GUID [SPACE_GUID...]",
		Short: "Share route with spaces",
		Long:  "Share a route with one or more spaces",
		Args:  cobra.MinimumNArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			routeIdentifier := args[0]
			spaceGUIDs := args[1:]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			route, err := findRouteByIdentifierOrURL(ctx, client, routeIdentifier)
			if err != nil {
				return err
			}

			relationship, err := client.Routes().ShareWithSpace(ctx, route.GUID, spaceGUIDs)
			if err != nil {
				return fmt.Errorf("sharing route: %w", err)
			}

			return renderRouteShareResult(relationship, route.URL, spaceGUIDs)
		},
	}
}

func findRouteByIdentifierOrURL(ctx context.Context, client capi.Client, routeIdentifier string) (*capi.Route, error) {
	// Try to find route by GUID first
	route, err := client.Routes().Get(ctx, routeIdentifier)
	if err == nil {
		return route, nil
	}

	// Try to find by URL
	params := capi.NewQueryParams()

	routes, err := client.Routes().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	for _, r := range routes.Resources {
		if r.URL == routeIdentifier {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("route '%s': %w", routeIdentifier, ErrRouteNotFound)
}

func renderRouteShareResult(relationship *capi.ToManyRelationship, routeURL string, spaceGUIDs []string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(relationship)
		if err != nil {
			return fmt.Errorf("failed to encode relationship: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(relationship)
		if err != nil {
			return fmt.Errorf("failed to encode relationship: %w", err)
		}

		return nil
	default:
		_, _ = fmt.Fprintf(os.Stdout, "✓ Route '%s' shared with %d space(s)\n", routeURL, len(spaceGUIDs))

		for _, spaceGUID := range spaceGUIDs {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", spaceGUID)
		}
	}

	return nil
}

func newRoutesUnshareCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unshare ROUTE_GUID_OR_URL SPACE_GUID",
		Short: "Unshare route from space",
		Long:  "Remove sharing of a route from a space",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			routeIdentifier := args[0]
			spaceGUID := args[1]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Try to find route by GUID first, then by URL
			route, err := client.Routes().Get(ctx, routeIdentifier)
			if err != nil {
				// Try to find by URL
				params := capi.NewQueryParams()

				routes, err := client.Routes().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to list routes: %w", err)
				}

				for _, r := range routes.Resources {
					if r.URL == routeIdentifier {
						route = &r

						break
					}
				}

				if route == nil {
					return fmt.Errorf("route '%s': %w", routeIdentifier, ErrRouteNotFound)
				}
			}

			// Unshare from space
			err = client.Routes().UnshareFromSpace(ctx, route.GUID, spaceGUID)
			if err != nil {
				return fmt.Errorf("unsharing route: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "✓ Route '%s' unshared from space '%s'\n", route.URL, spaceGUID)

			return nil
		},
	}
}

func newRoutesTransferCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "transfer ROUTE_GUID_OR_URL SPACE_GUID",
		Short: "Transfer route ownership to space",
		Long:  "Transfer ownership of a route to a different space",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			routeIdentifier := args[0]
			spaceGUID := args[1]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Try to find route by GUID first, then by URL
			route, err := client.Routes().Get(ctx, routeIdentifier)
			if err != nil {
				// Try to find by URL
				params := capi.NewQueryParams()

				routes, err := client.Routes().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to list routes: %w", err)
				}

				for _, r := range routes.Resources {
					if r.URL == routeIdentifier {
						route = &r

						break
					}
				}

				if route == nil {
					return fmt.Errorf("route '%s': %w", routeIdentifier, ErrRouteNotFound)
				}
			}

			// Transfer ownership
			updatedRoute, err := client.Routes().TransferOwnership(ctx, route.GUID, spaceGUID)
			if err != nil {
				return fmt.Errorf("transferring route ownership: %w", err)
			}

			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(updatedRoute)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(updatedRoute)
			default:
				_, _ = fmt.Fprintf(os.Stdout, "✓ Route '%s' ownership transferred to space '%s'\n", route.URL, spaceGUID)
			}

			return nil
		},
	}
}

func newRoutesListSharedCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list-shared ROUTE_GUID_OR_URL",
		Short: "List spaces that a route is shared with",
		Long:  "List all spaces that a route is shared with",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			routeIdentifier := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			route, err := findRouteByIdentifierOrURL(ctx, client, routeIdentifier)
			if err != nil {
				return err
			}

			sharedSpaces, err := client.Routes().ListSharedSpaces(ctx, route.GUID)
			if err != nil {
				return fmt.Errorf("listing shared spaces: %w", err)
			}

			return renderSharedSpaces(ctx, client, route.URL, sharedSpaces)
		},
	}
}

func renderSharedSpaces(ctx context.Context, client capi.Client, routeURL string, sharedSpaces *capi.ListResponse[capi.Space]) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(sharedSpaces)
		if err != nil {
			return fmt.Errorf("failed to encode shared spaces: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(sharedSpaces)
		if err != nil {
			return fmt.Errorf("failed to encode shared spaces: %w", err)
		}

		return nil
	default:
		return renderSharedSpacesTable(ctx, client, routeURL, sharedSpaces.Resources)
	}
}

func renderSharedSpacesTable(ctx context.Context, client capi.Client, routeURL string, spaces []capi.Space) error {
	if len(spaces) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Route '%s' is not shared with any spaces\n", routeURL)

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Space Name", "Space GUID", "Organization")

	for _, space := range spaces {
		orgName := getOrganizationName(ctx, client, space)
		_ = table.Append(space.Name, space.GUID, orgName)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Shared spaces for route '%s':\n\n", routeURL)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render shared spaces table: %w", err)
	}

	return nil
}

func getOrganizationName(ctx context.Context, client capi.Client, space capi.Space) string {
	if space.Relationships.Organization.Data == nil {
		return ""
	}

	// Try to get org name - if it fails, just use GUID
	org, err := client.Organizations().Get(ctx, space.Relationships.Organization.Data.GUID)
	if err == nil {
		return org.Name
	}

	return space.Relationships.Organization.Data.GUID
}
