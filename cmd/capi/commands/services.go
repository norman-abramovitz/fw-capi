package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewServicesCommand creates the services command group.
func NewServicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "services",
		Aliases: []string{"service"},
		Short:   "Manage services",
		Long:    "List and manage Cloud Foundry services and service instances",
	}

	cmd.AddCommand(newServicesListCommand())
	cmd.AddCommand(newServicesGetCommand())
	cmd.AddCommand(newServicesCreateCommand())
	cmd.AddCommand(newServicesUpdateCommand())
	cmd.AddCommand(newServicesDeleteCommand())
	cmd.AddCommand(newServicesBindCommand())
	cmd.AddCommand(newServicesUnbindCommand())
	cmd.AddCommand(newServicesRenameCommand())
	cmd.AddCommand(newServicesShareCommand())
	cmd.AddCommand(newServicesUnshareCommand())
	cmd.AddCommand(newServicesListBindingsCommand())
	cmd.AddCommand(newServicesBrokersCommand())
	cmd.AddCommand(newServicesOfferingsCommand())
	cmd.AddCommand(newServicesPlansCommand())

	return cmd
}

// serviceListConfig holds configuration for listing services.
type serviceListConfig struct {
	spaceName string
	allPages  bool
	perPage   int
}

// setupServiceListParams configures query parameters for service listing.
func setupServiceListParams(ctx context.Context, client capi.Client, config *serviceListConfig) (*capi.QueryParams, error) {
	params := capi.NewQueryParams()
	params.PerPage = config.perPage

	// Filter by space if specified
	if config.spaceName != "" {
		spaceParams := capi.NewQueryParams()
		spaceParams.WithFilter("names", config.spaceName)

		// Add org filter if targeted
		if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
			spaceParams.WithFilter("organization_guids", orgGUID)
		}

		spaces, err := client.Spaces().List(ctx, spaceParams)
		if err != nil {
			return nil, fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return nil, fmt.Errorf("space '%s': %w", config.spaceName, ErrSpaceNotFound)
		}

		params.WithFilter("space_guids", spaces.Resources[0].GUID)
	} else if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		// Use targeted space
		params.WithFilter("space_guids", spaceGUID)
	}

	return params, nil
}

// fetchAllServicePages retrieves all pages of services if requested.
func fetchAllServicePages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.ServiceInstance, *capi.Pagination, error) {
	services, err := client.ServiceInstances().List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list service instances: %w", err)
	}

	allServices := services.Resources

	if allPages && services.Pagination.TotalPages > 1 {
		for page := 2; page <= services.Pagination.TotalPages; page++ {
			params.Page = page

			moreServices, err := client.ServiceInstances().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allServices = append(allServices, moreServices.Resources...)
		}
	}

	return allServices, &services.Pagination, nil
}

// enrichServiceDetails fetches additional details for service instances.
func enrichServiceDetails(ctx context.Context, client capi.Client, service *capi.ServiceInstance) (string, string, int) {
	// Get service plan and offering details if available
	planName, offeringName := getServicePlanAndOfferingNames(ctx, client, service)

	// Get bound apps count
	bindingsParams := capi.NewQueryParams()
	bindingsParams.WithFilter("service_instance_guids", service.GUID)

	bindings, _ := client.ServiceCredentialBindings().List(ctx, bindingsParams)

	boundApps := 0
	if bindings != nil {
		boundApps = len(bindings.Resources)
	}

	return planName, offeringName, boundApps
}

// getServiceOfferingName retrieves the service offering name for a service plan.
func getServiceOfferingName(ctx context.Context, client capi.Client, plan any) string {
	// Use reflection to access plan fields since we don't know the exact type
	planValue := reflect.ValueOf(plan)
	if planValue.Kind() == reflect.Pointer {
		planValue = planValue.Elem()
	}

	relationshipsField := planValue.FieldByName("Relationships")
	if !relationshipsField.IsValid() {
		return ""
	}

	serviceOfferingField := relationshipsField.FieldByName("ServiceOffering")
	if !serviceOfferingField.IsValid() {
		return ""
	}

	dataField := serviceOfferingField.FieldByName("Data")
	if !dataField.IsValid() || dataField.IsNil() {
		return ""
	}

	guidField := dataField.Elem().FieldByName("GUID")
	if !guidField.IsValid() {
		return ""
	}

	offering, _ := client.ServiceOfferings().Get(ctx, guidField.String())
	if offering == nil {
		return ""
	}

	offeringValue := reflect.ValueOf(offering)
	if offeringValue.Kind() == reflect.Pointer {
		offeringValue = offeringValue.Elem()
	}

	nameField := offeringValue.FieldByName("Name")
	if !nameField.IsValid() {
		return ""
	}

	return nameField.String()
}

// getServicePlanAndOfferingNames retrieves both service plan and offering names.
func getServicePlanAndOfferingNames(ctx context.Context, client capi.Client, service *capi.ServiceInstance) (string, string) {
	if service.Relationships.ServicePlan == nil || service.Relationships.ServicePlan.Data == nil {
		return "", ""
	}

	plan, _ := client.ServicePlans().Get(ctx, service.Relationships.ServicePlan.Data.GUID)
	if plan == nil {
		return "", ""
	}

	planName := plan.Name
	offeringName := getServiceOfferingName(ctx, client, plan)

	return planName, offeringName
}

// outputServiceList renders the service list in the requested format.
func outputServiceList(ctx context.Context, client capi.Client, services []capi.ServiceInstance, pagination *capi.Pagination, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(services)
		if err != nil {
			return fmt.Errorf("encoding services to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(services)
		if err != nil {
			return fmt.Errorf("encoding services to YAML: %w", err)
		}

		return nil
	default:
		return renderServiceTable(ctx, client, services, pagination, allPages)
	}
}

// renderServiceTable renders services in table format.
func renderServiceTable(ctx context.Context, client capi.Client, services []capi.ServiceInstance, pagination *capi.Pagination, allPages bool) error {
	if len(services) == 0 {
		_, _ = os.Stdout.WriteString("No service instances found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Service", "Plan", "Bound Apps", "State", "Type")

	for _, service := range services {
		planName, offeringName, boundApps := enrichServiceDetails(ctx, client, &service)

		state := Ready
		if service.LastOperation != nil {
			state = service.LastOperation.State
		}

		_ = table.Append(service.Name, offeringName, planName, strconv.Itoa(boundApps), state, service.Type)
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newServicesListCommand() *cobra.Command {
	config := &serviceListConfig{}

	cmd := &cobra.Command{
		Use:   List,
		Short: "List service instances",
		Long:  "List all service instances the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Setup query parameters
			params, err := setupServiceListParams(ctx, client, config)
			if err != nil {
				return err
			}

			// Fetch services (all pages if requested)
			services, pagination, err := fetchAllServicePages(ctx, client, params, config.allPages)
			if err != nil {
				return err
			}

			// Output results in requested format
			return outputServiceList(ctx, client, services, pagination, config.allPages)
		},
	}

	cmd.Flags().StringVarP(&config.spaceName, spaceKey, "s", "", "filter by space name")
	cmd.Flags().BoolVar(&config.allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&config.perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

func newServicesGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get SERVICE_NAME_OR_GUID",
		Short: "Get service instance details",
		Long:  "Display detailed information about a specific service instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Resolve service instance
			service, err := resolveServiceInstance(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// Output results
			return renderServiceOutput(ctx, client, service)
		},
	}
}

// resolveServiceInstance resolves service by name or GUID.
func resolveServiceInstance(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.ServiceInstance, error) {
	servicesClient := client.ServiceInstances()

	// Try to get by GUID first
	service, err := servicesClient.Get(ctx, nameOrGUID)
	if err == nil {
		return service, nil
	}

	// Try by name in targeted space
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		params.WithFilter("space_guids", spaceGUID)
	}

	services, err := servicesClient.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find service instance: %w", err)
	}

	if len(services.Resources) == 0 {
		return nil, fmt.Errorf("service instance '%s': %w", nameOrGUID, ErrServiceInstanceNotFound)
	}

	return &services.Resources[0], nil
}

// handleServiceBindingResult processes and displays the result of service binding operations.
func handleServiceBindingResult(result any, serviceName, appName, operation string) {
	if binding, ok := result.(*capi.ServiceCredentialBinding); ok {
		_, _ = fmt.Fprintf(os.Stdout, "Successfully %s service instance '%s' %s application '%s'\n", operation, serviceName, getBindingPreposition(operation), appName)
		_, _ = fmt.Fprintf(os.Stdout, "  Binding GUID: %s\n", binding.GUID)
	} else if job, ok := result.(*capi.Job); ok {
		_, _ = fmt.Fprintf(os.Stdout, "Successfully initiated %s of service instance '%s' %s application '%s'\n", operation, serviceName, getBindingPreposition(operation), appName)
		_, _ = fmt.Fprintf(os.Stdout, "  Job GUID: %s\n", job.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "  Monitor with: capi jobs get %s\n", job.GUID)
	}
}

// getBindingPreposition returns the appropriate preposition for binding operations.
func getBindingPreposition(operation string) string {
	if operation == "unbound" {
		return "from"
	}

	return "to"
}

// handleJobResult processes and displays the result of job-based operations.
func handleJobResult(job *capi.Job, entityName, targetName, action string, preposition string) {
	if job != nil {
		_, _ = fmt.Fprintf(os.Stdout, "%s %s '%s' %s %s '%s'... (job: %s)\n",
			cases.Title(language.English).String(action+"ing"),
			getEntityType(action), entityName, preposition, getTargetType(action), targetName, job.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "Monitor with: capi jobs get %s\n", job.GUID)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Successfully %s %s '%s' %s %s '%s'\n",
			action, getEntityType(action), entityName, preposition, getTargetType(action), targetName)
	}
}

// getEntityType returns the entity type based on the action.
func getEntityType(action string) string {
	if strings.Contains(action, "service") {
		return "service instance"
	}

	return "entity"
}

// getTargetType returns the target type based on the action.
func getTargetType(action string) string {
	if strings.Contains(action, "bind") {
		return "application"
	}

	return "target"
}

// fetchAllBindingPages retrieves all pages of service credential bindings if requested.
func fetchAllBindingPages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.ServiceCredentialBinding, *capi.Pagination, error) {
	bindings, err := client.ServiceCredentialBindings().List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list service bindings: %w", err)
	}

	allBindings := bindings.Resources

	if allPages && bindings.Pagination.TotalPages > 1 {
		for page := 2; page <= bindings.Pagination.TotalPages; page++ {
			params.Page = page

			moreBindings, err := client.ServiceCredentialBindings().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allBindings = append(allBindings, moreBindings.Resources...)
		}
	}

	return allBindings, &bindings.Pagination, nil
}

// renderBindingsOutput renders service credential bindings in the specified format.
func renderBindingsOutput(ctx context.Context, client capi.Client, bindings []capi.ServiceCredentialBinding, serviceName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(bindings)
	case OutputFormatYAML:
		return StandardYAMLRenderer(bindings)
	default:
		return renderBindingsTable(ctx, client, bindings, serviceName)
	}
}

// renderBindingsTable renders service credential bindings as a table.
func renderBindingsTable(ctx context.Context, client capi.Client, bindings []capi.ServiceCredentialBinding, serviceName string) error {
	if len(bindings) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No bindings found for service instance '%s'\n", serviceName)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Bindings for service instance '%s':\n\n", serviceName)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Type", "App", "State", "Created")

	for _, binding := range bindings {
		appName := ""

		if binding.Relationships.App != nil && binding.Relationships.App.Data != nil {
			app, _ := client.Apps().Get(ctx, binding.Relationships.App.Data.GUID)
			if app != nil {
				appName = app.Name
			}
		}

		state := Ready
		if binding.LastOperation != nil {
			state = binding.LastOperation.State
		}

		_ = table.Append(binding.Name, binding.Type, appName, state, binding.CreatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	return nil
}

// renderServiceOutput renders service instance in the specified format.
func renderServiceOutput(ctx context.Context, client capi.Client, service *capi.ServiceInstance) error {
	output := viper.GetString("output")

	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(service)
	case OutputFormatYAML:
		return StandardYAMLRenderer(service)
	default:
		return renderServiceDetailsTable(ctx, client, service)
	}
}

// renderServiceDetailsTable renders service instance as a table.
func renderServiceDetailsTable(ctx context.Context, client capi.Client, service *capi.ServiceInstance) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	// Basic service information
	addBasicServiceInfo(table, service)

	// Service plan and offering details
	addServicePlanInfo(ctx, client, table, service)

	// Space information
	addServiceSpaceInfo(ctx, client, table, service)

	// Last operation
	addServiceLastOperationInfo(table, service)

	// Additional metadata
	addServiceMetadata(table, service)

	// User-provided specific fields
	addUserProvidedServiceInfo(table, service)

	_, _ = fmt.Fprintf(os.Stdout, "Service Instance: %s\n\n", service.Name)

	_ = table.Render()

	return nil
}

// addBasicServiceInfo adds basic service information to table.
func addBasicServiceInfo(table *tablewriter.Table, service *capi.ServiceInstance) {
	_ = table.Append("Name", service.Name)
	_ = table.Append("GUID", service.GUID)
	_ = table.Append("Type", service.Type)
	_ = table.Append("Created", service.CreatedAt.Format(TimeFormatDisplay))
	_ = table.Append("Updated", service.UpdatedAt.Format(TimeFormatDisplay))
}

// addServicePlanInfo adds service plan and offering information.
func addServicePlanInfo(ctx context.Context, client capi.Client, table *tablewriter.Table, service *capi.ServiceInstance) {
	if service.Relationships.ServicePlan == nil || service.Relationships.ServicePlan.Data == nil {
		return
	}

	plan, _ := client.ServicePlans().Get(ctx, service.Relationships.ServicePlan.Data.GUID)
	if plan == nil {
		return
	}

	_ = table.Append("Plan", plan.Name)

	if plan.Relationships.ServiceOffering.Data != nil {
		offering, _ := client.ServiceOfferings().Get(ctx, plan.Relationships.ServiceOffering.Data.GUID)
		if offering != nil {
			_ = table.Append("Service", offering.Name)
		}
	}
}

// addServiceSpaceInfo adds space information.
func addServiceSpaceInfo(ctx context.Context, client capi.Client, table *tablewriter.Table, service *capi.ServiceInstance) {
	if service.Relationships.Space.Data == nil {
		return
	}

	space, _ := client.Spaces().Get(ctx, service.Relationships.Space.Data.GUID)
	if space != nil {
		_ = table.Append("Space", space.Name)
	}
}

// addServiceLastOperationInfo adds last operation information.
func addServiceLastOperationInfo(table *tablewriter.Table, service *capi.ServiceInstance) {
	if service.LastOperation == nil {
		return
	}

	_ = table.Append("Last Operation Type", service.LastOperation.Type)
	_ = table.Append("Last Operation State", service.LastOperation.State)
	_ = table.Append("Last Operation Description", service.LastOperation.Description)

	if service.LastOperation.UpdatedAt != nil {
		_ = table.Append("Last Operation Updated", service.LastOperation.UpdatedAt.Format(TimeFormatDisplay))
	}
}

// addServiceMetadata adds tags and dashboard URL.
func addServiceMetadata(table *tablewriter.Table, service *capi.ServiceInstance) {
	if len(service.Tags) > 0 {
		_ = table.Append("Tags", strings.Join(service.Tags, ", "))
	}

	if service.DashboardURL != nil {
		_ = table.Append("Dashboard", *service.DashboardURL)
	}
}

// addUserProvidedServiceInfo adds user-provided service specific fields.
func addUserProvidedServiceInfo(table *tablewriter.Table, service *capi.ServiceInstance) {
	if service.Type != UserProvided {
		return
	}

	if service.SyslogDrainURL != nil {
		_ = table.Append("Syslog Drain", *service.SyslogDrainURL)
	}

	if service.RouteServiceURL != nil {
		_ = table.Append("Route Service", *service.RouteServiceURL)
	}
}

// serviceCreateConfig holds all the configuration for creating a service.
type serviceCreateConfig struct {
	serviceName     string
	planName        string
	spaceName       string
	tags            []string
	parameters      map[string]string
	syslogDrainURL  string
	routeServiceURL string
	credentials     map[string]string
	userProvided    bool
}

// resolveServicePlanGUID finds the service plan GUID for managed services.
func resolveServicePlanGUID(ctx context.Context, client capi.Client, planName, serviceOfferingArg string) (string, error) {
	var serviceOfferingGUID string

	// If service offering is provided, find it first
	if serviceOfferingArg != "" {
		offeringParams := capi.NewQueryParams()
		offeringParams.WithFilter("names", serviceOfferingArg)

		offerings, err := client.ServiceOfferings().List(ctx, offeringParams)
		if err != nil {
			return "", fmt.Errorf("failed to find service offering: %w", err)
		}

		if len(offerings.Resources) == 0 {
			return "", fmt.Errorf("service offering '%s': %w", serviceOfferingArg, ErrServiceOfferingNotFound)
		}

		serviceOfferingGUID = offerings.Resources[0].GUID
	}

	// Find service plan by name
	planParams := capi.NewQueryParams()
	planParams.WithFilter("names", planName)

	// If service offering was specified, filter plans by it
	if serviceOfferingGUID != "" {
		planParams.WithFilter("service_offering_guids", serviceOfferingGUID)
	}

	plans, err := client.ServicePlans().List(ctx, planParams)
	if err != nil {
		return "", fmt.Errorf("failed to find service plan: %w", err)
	}

	if len(plans.Resources) == 0 {
		if serviceOfferingGUID != "" {
			return "", fmt.Errorf("service plan '%s' not found for the specified service offering: %w", planName, ErrServicePlanNotFound)
		}

		return "", fmt.Errorf("service plan '%s': %w", planName, ErrServicePlanNotFound)
	}

	return plans.Resources[0].GUID, nil
}

// buildUserProvidedServiceRequest creates a request for user-provided services.
func buildUserProvidedServiceRequest(config *serviceCreateConfig, spaceGUID string) *capi.ServiceInstanceCreateRequest {
	req := &capi.ServiceInstanceCreateRequest{
		Name: config.serviceName,
		Type: UserProvided,
		Relationships: capi.ServiceInstanceRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{GUID: spaceGUID},
			},
		},
		Tags: config.tags,
	}

	// Convert credentials map
	if len(config.credentials) > 0 {
		credMap := make(map[string]any)
		for k, v := range config.credentials {
			credMap[k] = v
		}

		req.Credentials = credMap
	}

	if config.syslogDrainURL != "" {
		req.SyslogDrainURL = &config.syslogDrainURL
	}

	if config.routeServiceURL != "" {
		req.RouteServiceURL = &config.routeServiceURL
	}

	return req
}

// buildManagedServiceRequest creates a request for managed services.
func buildManagedServiceRequest(config *serviceCreateConfig, spaceGUID, planGUID string) *capi.ServiceInstanceCreateRequest {
	req := &capi.ServiceInstanceCreateRequest{
		Name: config.serviceName,
		Type: "managed",
		Relationships: capi.ServiceInstanceRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{GUID: spaceGUID},
			},
			ServicePlan: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: planGUID},
			},
		},
		Tags: config.tags,
	}

	// Convert parameters
	if len(config.parameters) > 0 {
		paramMap := make(map[string]any)
		for k, v := range config.parameters {
			paramMap[k] = v
		}

		req.Parameters = paramMap
	}

	return req
}

// handleServiceCreationResult processes and displays the result of service creation.
func handleServiceCreationResult(result any, serviceName string, userProvided bool) {
	if userProvided {
		if service, ok := result.(*capi.ServiceInstance); ok {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully created user-provided service instance '%s'\n", service.Name)
			_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", service.GUID)
		}
	} else {
		if job, ok := result.(*capi.Job); ok {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully initiated creation of managed service instance '%s'\n", serviceName)
			_, _ = fmt.Fprintf(os.Stdout, "  Job GUID: %s\n", job.GUID)
			_, _ = fmt.Fprintf(os.Stdout, "  Monitor with: capi jobs get %s\n", job.GUID)
		}
	}
}

func newServicesCreateCommand() *cobra.Command {
	config := &serviceCreateConfig{}

	cmd := &cobra.Command{
		Use:   "create [service-offering] [flags]",
		Short: "Create a service instance",
		Long:  "Create a new service instance (managed or user-provided)\n\nFor managed services, you can optionally specify the service offering name as the first argument.\nIf provided, it will filter the plan search to that specific service offering.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeServiceCreate(cmd, args, config)
		},
	}

	cmd.Flags().StringVarP(&config.serviceName, "name", "n", "", "service instance name (required)")
	cmd.Flags().StringVarP(&config.planName, "plan", "p", "", "service plan name (required for managed services)")
	cmd.Flags().StringVarP(&config.spaceName, spaceKey, "s", "", "space name (defaults to targeted space)")
	cmd.Flags().StringArrayVar(&config.tags, "tags", nil, "tags for the service instance")
	cmd.Flags().StringToStringVar(&config.parameters, "parameters", nil, "parameters for managed service instances (key=value)")
	cmd.Flags().StringVar(&config.syslogDrainURL, "syslog-drain-url", "", "syslog drain URL for user-provided services")
	cmd.Flags().StringVar(&config.routeServiceURL, "route-service-url", "", "route service URL for user-provided services")
	cmd.Flags().StringToStringVar(&config.credentials, "credentials", nil, "credentials for user-provided services (key=value)")
	cmd.Flags().BoolVar(&config.userProvided, UserProvided, false, "create a user-provided service instance")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

// executeServiceCreate handles the service creation logic.
func executeServiceCreate(cmd *cobra.Command, args []string, config *serviceCreateConfig) error {
	if config.serviceName == "" {
		return ErrServiceInstanceNameRequired
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Find space GUID
	spaceGUID, err := resolveSpaceGUIDForServices(ctx, client, config.spaceName)
	if err != nil {
		return err
	}

	createReq, err := prepareServiceCreateRequest(ctx, client, config, spaceGUID, args)
	if err != nil {
		return err
	}

	result, err := client.ServiceInstances().Create(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create service instance: %w", err)
	}

	handleServiceCreationResult(result, config.serviceName, config.userProvided)

	return nil
}

// prepareServiceCreateRequest prepares the service creation request.
func prepareServiceCreateRequest(ctx context.Context, client capi.Client, config *serviceCreateConfig, spaceGUID string, args []string) (*capi.ServiceInstanceCreateRequest, error) {
	if config.userProvided {
		return buildUserProvidedServiceRequest(config, spaceGUID), nil
	}

	if config.planName == "" {
		return nil, ErrServicePlanRequiredForManaged
	}

	serviceOfferingArg := ""
	if len(args) > 0 {
		serviceOfferingArg = args[0]
	}

	planGUID, err := resolveServicePlanGUID(ctx, client, config.planName, serviceOfferingArg)
	if err != nil {
		return nil, err
	}

	return buildManagedServiceRequest(config, spaceGUID, planGUID), nil
}

func newServicesUpdateCommand() *cobra.Command {
	var (
		newName         string
		tags            []string
		parameters      map[string]string
		syslogDrainURL  string
		routeServiceURL string
		credentials     map[string]string
		planName        string
	)

	cmd := &cobra.Command{
		Use:   "update SERVICE_NAME_OR_GUID",
		Short: "Update a service instance",
		Long:  "Update an existing service instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeServiceUpdate(cmd, args[0], newName, tags, credentials,
				syslogDrainURL, routeServiceURL, parameters, planName)
		},
	}

	cmd.Flags().StringVar(&newName, "name", "", "new service instance name")
	cmd.Flags().StringArrayVar(&tags, "tags", nil, "tags for the service instance")
	cmd.Flags().StringToStringVar(&parameters, "parameters", nil, "parameters for managed service instances (key=value)")
	cmd.Flags().StringVar(&syslogDrainURL, "syslog-drain-url", "", "syslog drain URL for user-provided services")
	cmd.Flags().StringVar(&routeServiceURL, "route-service-url", "", "route service URL for user-provided services")
	cmd.Flags().StringToStringVar(&credentials, "credentials", nil, "credentials for user-provided services (key=value)")
	cmd.Flags().StringVarP(&planName, "plan", "p", "", "new service plan name for managed services")

	return cmd
}

// executeServiceUpdate handles the service update logic.
func executeServiceUpdate(cmd *cobra.Command, nameOrGUID string, newName string, tags []string,
	credentials map[string]string, syslogDrainURL, routeServiceURL string,
	parameters map[string]string, planName string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	servicesClient := client.ServiceInstances()

	// Find service instance
	service, err := findServiceInstanceForUpdate(ctx, servicesClient, nameOrGUID)
	if err != nil {
		return err
	}

	// Build update request based on service type
	updateReq, err := buildServiceUpdateRequest(cmd, newName, tags, credentials,
		syslogDrainURL, routeServiceURL, parameters, planName, service.Type, client, ctx)
	if err != nil {
		return err
	}

	result, err := servicesClient.Update(ctx, service.GUID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update service instance: %w", err)
	}

	handleServiceUpdateResult(result, service.Type, service.Name)

	return nil
}

// findServiceInstanceForUpdate finds a service instance by name or GUID.
func findServiceInstanceForUpdate(ctx context.Context, servicesClient capi.ServiceInstancesClient, nameOrGUID string) (*capi.ServiceInstance, error) {
	service, err := servicesClient.Get(ctx, nameOrGUID)
	if err == nil {
		return service, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	// Filter by space if targeted
	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		params.WithFilter("space_guids", spaceGUID)
	}

	services, err := servicesClient.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find service instance: %w", err)
	}

	if len(services.Resources) == 0 {
		return nil, fmt.Errorf("service instance '%s': %w", nameOrGUID, ErrServiceInstanceNotFound)
	}

	return &services.Resources[0], nil
}

// buildServiceUpdateRequest builds the update request based on service type.
func buildServiceUpdateRequest(cmd *cobra.Command, newName string, tags []string,
	credentials map[string]string, syslogDrainURL, routeServiceURL string,
	parameters map[string]string, planName string, serviceType string, client capi.Client, ctx context.Context) (*capi.ServiceInstanceUpdateRequest, error) {
	updateReq := &capi.ServiceInstanceUpdateRequest{}

	if newName != "" {
		updateReq.Name = &newName
	}

	if len(tags) > 0 {
		updateReq.Tags = tags
	}

	if serviceType == UserProvided {
		return buildUserProvidedUpdateRequest(updateReq, cmd, credentials, syslogDrainURL, routeServiceURL), nil
	}

	return buildManagedServiceUpdateRequest(updateReq, parameters, planName, client, ctx)
}

// buildUserProvidedUpdateRequest configures update request for user-provided services.
func buildUserProvidedUpdateRequest(updateReq *capi.ServiceInstanceUpdateRequest, cmd *cobra.Command,
	credentials map[string]string, syslogDrainURL, routeServiceURL string) *capi.ServiceInstanceUpdateRequest {
	if len(credentials) > 0 {
		credMap := make(map[string]any)
		for k, v := range credentials {
			credMap[k] = v
		}

		updateReq.Credentials = credMap
	}

	if cmd.Flags().Changed("syslog-drain-url") {
		updateReq.SyslogDrainURL = &syslogDrainURL
	}

	if cmd.Flags().Changed("route-service-url") {
		updateReq.RouteServiceURL = &routeServiceURL
	}

	return updateReq
}

// buildManagedServiceUpdateRequest configures update request for managed services.
func buildManagedServiceUpdateRequest(updateReq *capi.ServiceInstanceUpdateRequest,
	parameters map[string]string, planName string, client capi.Client, ctx context.Context) (*capi.ServiceInstanceUpdateRequest, error) {
	if len(parameters) > 0 {
		paramMap := make(map[string]any)
		for k, v := range parameters {
			paramMap[k] = v
		}

		updateReq.Parameters = paramMap
	}

	if planName != "" {
		planParams := capi.NewQueryParams()
		planParams.WithFilter("names", planName)

		plans, err := client.ServicePlans().List(ctx, planParams)
		if err != nil {
			return nil, fmt.Errorf("failed to find service plan: %w", err)
		}

		if len(plans.Resources) == 0 {
			return nil, fmt.Errorf("service plan '%s': %w", planName, ErrServicePlanNotFound)
		}

		updateReq.Relationships = &capi.ServiceInstanceRelationships{
			ServicePlan: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: plans.Resources[0].GUID},
			},
		}
	}

	return updateReq, nil
}

// handleServiceUpdateResult processes the update result based on service type.
func handleServiceUpdateResult(result any, serviceType, serviceName string) {
	if serviceType == UserProvided {
		if updatedService, ok := result.(*capi.ServiceInstance); ok {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated user-provided service instance '%s'\n", updatedService.Name)
		}
	} else {
		if job, ok := result.(*capi.Job); ok {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully initiated update of managed service instance '%s'\n", serviceName)
			_, _ = fmt.Fprintf(os.Stdout, "  Job GUID: %s\n", job.GUID)
			_, _ = fmt.Fprintf(os.Stdout, "  Monitor with: capi jobs get %s\n", job.GUID)
		}
	}
}

func newServicesDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete SERVICE_NAME_OR_GUID",
		Short: "Delete a service instance",
		Long:  "Delete a service instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeServiceDelete(cmd, args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force deletion without confirmation")

	return cmd
}

// executeServiceDelete handles the service deletion logic.
func executeServiceDelete(cmd *cobra.Command, nameOrGUID string, force bool) error {
	if !force {
		if !confirmDeletion(nameOrGUID) {
			_, _ = os.Stdout.WriteString("Cancelled\n")

			return nil
		}
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	servicesClient := client.ServiceInstances()

	// Find service instance
	service, err := findServiceInstanceForUpdate(ctx, servicesClient, nameOrGUID)
	if err != nil {
		return err
	}

	// Delete service instance
	job, err := servicesClient.Delete(ctx, service.GUID)
	if err != nil {
		return fmt.Errorf("failed to delete service instance: %w", err)
	}

	handleServiceDeletionResult(job, service.Name)

	return nil
}

// confirmDeletion prompts the user to confirm deletion.
func confirmDeletion(nameOrGUID string) bool {
	_, _ = fmt.Fprintf(os.Stdout, "Really delete service instance '%s'? (y/N): ", nameOrGUID)

	var response string

	_, _ = fmt.Scanln(&response)

	return response == "y" || response == "Y"
}

// handleServiceDeletionResult handles the deletion result output.
func handleServiceDeletionResult(job *capi.Job, serviceName string) {
	if job != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Deleting service instance '%s'... (job: %s)\n", serviceName, job.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "Monitor with: capi jobs get %s\n", job.GUID)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Successfully deleted service instance '%s'\n", serviceName)
	}
}

func newServicesBindCommand() *cobra.Command {
	var (
		parameters  map[string]string
		bindingName string
	)

	cmd := &cobra.Command{
		Use:   "bind SERVICE_NAME_OR_GUID APP_NAME_OR_GUID",
		Short: "Bind a service instance to an application",
		Long:  "Create a service credential binding between a service instance and an application",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeServiceBind(cmd, args[0], args[1], bindingName, parameters)
		},
	}

	cmd.Flags().StringToStringVar(&parameters, "parameters", nil, "binding parameters (key=value)")
	cmd.Flags().StringVar(&bindingName, "name", "", "name for the service binding")

	return cmd
}

// executeServiceBind handles the service binding logic.
func executeServiceBind(cmd *cobra.Command, serviceNameOrGUID, appNameOrGUID, bindingName string, parameters map[string]string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve service instance and application
	service, err := resolveServiceInstance(ctx, client, serviceNameOrGUID)
	if err != nil {
		return err
	}

	appGUID, appName, err := resolveApp(ctx, client, appNameOrGUID)
	if err != nil {
		return err
	}

	// Create binding request
	createReq := buildBindingRequest(service.GUID, appGUID, bindingName, parameters)

	result, err := client.ServiceCredentialBindings().Create(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to bind service instance: %w", err)
	}

	handleServiceBindingResult(result, service.Name, appName, "bound")

	return nil
}

// buildBindingRequest builds the service credential binding request.
func buildBindingRequest(serviceGUID, appGUID, bindingName string, parameters map[string]string) *capi.ServiceCredentialBindingCreateRequest {
	createReq := &capi.ServiceCredentialBindingCreateRequest{
		Type: "app",
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{GUID: serviceGUID},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: appGUID},
			},
		},
	}

	if bindingName != "" {
		createReq.Name = &bindingName
	}

	if len(parameters) > 0 {
		paramMap := make(map[string]any)
		for k, v := range parameters {
			paramMap[k] = v
		}

		createReq.Parameters = paramMap
	}

	return createReq
}

func newServicesUnbindCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "unbind SERVICE_NAME_OR_GUID APP_NAME_OR_GUID",
		Short: "Unbind a service instance from an application",
		Long:  "Remove a service credential binding between a service instance and an application",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeServiceUnbind(cmd, args[0], args[1], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force unbinding without confirmation")

	return cmd
}

// executeServiceUnbind handles the service unbinding logic.
func executeServiceUnbind(cmd *cobra.Command, serviceNameOrGUID, appNameOrGUID string, force bool) error {
	if !force {
		if !confirmUnbinding(serviceNameOrGUID, appNameOrGUID) {
			_, _ = os.Stdout.WriteString("Cancelled\n")

			return nil
		}
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve service instance and application
	service, err := resolveServiceInstance(ctx, client, serviceNameOrGUID)
	if err != nil {
		return err
	}

	appGUID, appName, err := resolveApp(ctx, client, appNameOrGUID)
	if err != nil {
		return err
	}

	// Find and delete the binding
	binding, err := findServiceBinding(ctx, client, service.GUID, appGUID, service.Name, appName)
	if err != nil {
		return err
	}

	job, err := client.ServiceCredentialBindings().Delete(ctx, binding.GUID)
	if err != nil {
		return fmt.Errorf("failed to unbind service instance: %w", err)
	}

	handleJobResult(job, service.Name, appName, "unbind service", "from")

	return nil
}

// confirmUnbinding prompts the user to confirm unbinding.
func confirmUnbinding(serviceNameOrGUID, appNameOrGUID string) bool {
	_, _ = fmt.Fprintf(os.Stdout, "Really unbind service instance '%s' from application '%s'? (y/N): ", serviceNameOrGUID, appNameOrGUID)

	var response string

	_, _ = fmt.Scanln(&response)

	return response == "y" || response == "Y"
}

// findServiceBinding finds the binding between service and app.
func findServiceBinding(ctx context.Context, client capi.Client, serviceGUID, appGUID, serviceName, appName string) (*capi.ServiceCredentialBinding, error) {
	bindingParams := capi.NewQueryParams()
	bindingParams.WithFilter("service_instance_guids", serviceGUID)
	bindingParams.WithFilter("app_guids", appGUID)

	bindings, err := client.ServiceCredentialBindings().List(ctx, bindingParams)
	if err != nil {
		return nil, fmt.Errorf("failed to find service binding: %w", err)
	}

	if len(bindings.Resources) == 0 {
		return nil, fmt.Errorf("no binding found between service '%s' and app '%s': %w", serviceName, appName, ErrBindingNotFound)
	}

	return &bindings.Resources[0], nil
}

func newServicesRenameCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rename SERVICE_NAME_OR_GUID NEW_NAME",
		Short: "Rename a service instance",
		Long:  "Change the name of a service instance",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeServiceRename(cmd, args[0], args[1])
		},
	}
}

// executeServiceRename handles the service rename logic.
func executeServiceRename(cmd *cobra.Command, nameOrGUID, newName string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	servicesClient := client.ServiceInstances()

	// Find service instance
	service, err := findServiceInstanceForUpdate(ctx, servicesClient, nameOrGUID)
	if err != nil {
		return err
	}

	// Update with new name
	updateReq := &capi.ServiceInstanceUpdateRequest{
		Name: &newName,
	}

	result, err := servicesClient.Update(ctx, service.GUID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to rename service instance: %w", err)
	}

	handleRenameResult(result, service.Type, newName)

	return nil
}

// handleRenameResult handles the rename operation result.
func handleRenameResult(result any, serviceType, newName string) {
	if serviceType == UserProvided {
		// User-provided services return the service instance directly
		if updatedService, ok := result.(*capi.ServiceInstance); ok {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully renamed service instance to '%s'\n", updatedService.Name)
		}
	} else {
		// Managed services return a job
		if job, ok := result.(*capi.Job); ok {
			_, _ = fmt.Fprintf(os.Stdout, "Successfully initiated rename of service instance to '%s'\n", newName)
			_, _ = fmt.Fprintf(os.Stdout, "  Job GUID: %s\n", job.GUID)
			_, _ = fmt.Fprintf(os.Stdout, "  Monitor with: capi jobs get %s\n", job.GUID)
		}
	}
}

// findServiceInstanceByNameOrGUID finds a service instance by name or GUID.
func findServiceInstanceByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.ServiceInstance, error) {
	servicesClient := client.ServiceInstances()

	// Try direct GUID lookup first
	service, err := servicesClient.Get(ctx, nameOrGUID)
	if err == nil {
		return service, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	// Filter by space if targeted
	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		params.WithFilter("space_guids", spaceGUID)
	}

	services, err := servicesClient.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find service instance: %w", err)
	}

	if len(services.Resources) == 0 {
		return nil, fmt.Errorf("service instance '%s': %w", nameOrGUID, ErrServiceInstanceNotFound)
	}

	return &services.Resources[0], nil
}

// resolveSpacesToShareWith resolves space names to relationships.
func resolveSpacesToShareWith(ctx context.Context, client capi.Client, spaceNames []string) ([]capi.Relationship, error) {
	spaceRelationships := make([]capi.Relationship, 0, len(spaceNames))

	for _, spaceName := range spaceNames {
		params := capi.NewQueryParams()
		params.WithFilter("names", spaceName)

		// Add org filter if targeted
		if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
			params.WithFilter("organization_guids", orgGUID)
		}

		spaces, err := client.Spaces().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to find space '%s': %w", spaceName, err)
		}

		if len(spaces.Resources) == 0 {
			return nil, fmt.Errorf("space '%s': %w", spaceName, ErrSpaceNotFound)
		}

		spaceRelationships = append(spaceRelationships, capi.Relationship{
			Data: &capi.RelationshipData{GUID: spaces.Resources[0].GUID},
		})
	}

	return spaceRelationships, nil
}

func newServicesShareCommand() *cobra.Command {
	var spaceNames []string

	cmd := &cobra.Command{
		Use:   "share SERVICE_NAME_OR_GUID",
		Short: "Share a service instance with other spaces",
		Long:  "Share a service instance with specified spaces",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(spaceNames) == 0 {
				return ErrAtLeastOneSpaceRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find service instance
			service, err := findServiceInstanceByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			// Resolve spaces to share with
			spaceRelationships, err := resolveSpacesToShareWith(ctx, client, spaceNames)
			if err != nil {
				return err
			}

			// Share with spaces
			shareReq := &capi.ServiceInstanceShareRequest{
				Data: spaceRelationships,
			}

			_, err = client.ServiceInstances().ShareWithSpaces(ctx, service.GUID, shareReq)
			if err != nil {
				return fmt.Errorf("failed to share service instance: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully shared service instance '%s' with spaces: %s\n",
				service.Name, strings.Join(spaceNames, ", "))

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&spaceNames, "spaces", "s", nil, "spaces to share with (required)")
	_ = cmd.MarkFlagRequired("spaces")

	return cmd
}

func newServicesUnshareCommand() *cobra.Command {
	var spaceName string

	cmd := &cobra.Command{
		Use:   "unshare SERVICE_NAME_OR_GUID",
		Short: "Unshare a service instance from a space",
		Long:  "Remove sharing of a service instance from a specified space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceName == "" {
				return ErrSpaceNameRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find service instance
			service, err := findServiceInstanceByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			// Find space to unshare from
			spaceGUID, err := resolveSpaceGUID(ctx, client, spaceName)
			if err != nil {
				return err
			}

			// Unshare from space
			err = client.ServiceInstances().UnshareFromSpace(ctx, service.GUID, spaceGUID)
			if err != nil {
				return fmt.Errorf("failed to unshare service instance: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully unshared service instance '%s' from space '%s'\n",
				service.Name, spaceName)

			return nil
		},
	}

	cmd.Flags().StringVarP(&spaceName, spaceKey, "s", "", "space to unshare from (required)")
	_ = cmd.MarkFlagRequired(spaceKey)

	return cmd
}

func newServicesListBindingsCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list-bindings SERVICE_NAME_OR_GUID",
		Short: "List service bindings for a service instance",
		Long:  "List all credential bindings for a service instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Resolve service instance
			service, err := resolveServiceInstance(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// List bindings for this service instance
			params := capi.NewQueryParams()
			params.PerPage = perPage
			params.WithFilter("service_instance_guids", service.GUID)

			allBindings, _, err := fetchAllBindingPages(ctx, client, params, allPages)
			if err != nil {
				return err
			}

			return renderBindingsOutput(ctx, client, allBindings, service.Name)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

func newServicesBrokersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "brokers",
		Aliases: []string{"broker"},
		Short:   "Manage service brokers",
		Long:    "List and manage Cloud Foundry service brokers",
	}

	cmd.AddCommand(newServicesBrokersListCommand())
	cmd.AddCommand(newServicesBrokersGetCommand())

	return cmd
}

// renderServiceBrokersList renders the service brokers list in the specified output format
// renderServiceBrokersList renders the service brokers list in the specified output format
// renderServiceBrokersList renders the service brokers list in the specified output format.
func renderServiceBrokersList(ctx context.Context, client capi.Client, brokers *capi.ListResponse[capi.ServiceBroker], output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(brokers.Resources)
		if err != nil {
			return fmt.Errorf("encoding brokers to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(brokers.Resources)
		if err != nil {
			return fmt.Errorf("encoding brokers to YAML: %w", err)
		}

		return nil
	default:
		if len(brokers.Resources) == 0 {
			_, _ = os.Stdout.WriteString("No service brokers found\n")

			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Name", "URL", "GUID", "Space", "Created")

		for _, broker := range brokers.Resources {
			spaceName := "platform"

			if broker.Relationships.Space != nil && broker.Relationships.Space.Data != nil {
				space, _ := client.Spaces().Get(ctx, broker.Relationships.Space.Data.GUID)
				if space != nil {
					spaceName = space.Name
				}
			}

			_ = table.Append(broker.Name, broker.URL, broker.GUID, spaceName, broker.CreatedAt.Format("2006-01-02"))
		}

		_ = table.Render()
	}

	return nil
}

func newServicesBrokersListCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List service brokers",
		Long:  "List all service brokers",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := capi.NewQueryParams()
			params.PerPage = perPage

			brokers, err := client.ServiceBrokers().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list service brokers: %w", err)
			}

			return renderServiceBrokersList(ctx, client, brokers, viper.GetString("output"))
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

// findServiceBrokerByNameOrGUID finds a service broker by name or GUID.
func findServiceBrokerByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.ServiceBroker, error) {
	// Try to get by GUID first
	broker, err := client.ServiceBrokers().Get(ctx, nameOrGUID)
	if err == nil {
		return broker, nil
	}

	// If not found by GUID, try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	brokers, err := client.ServiceBrokers().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find service broker: %w", err)
	}

	if len(brokers.Resources) == 0 {
		return nil, fmt.Errorf("service broker '%s': %w", nameOrGUID, ErrServiceBrokerNotFound)
	}

	return &brokers.Resources[0], nil
}

// renderServiceBrokerDetails renders service broker details in the specified output format.
func renderServiceBrokerDetails(ctx context.Context, client capi.Client, broker *capi.ServiceBroker, output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(broker)
		if err != nil {
			return fmt.Errorf("encoding broker to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(broker)
		if err != nil {
			return fmt.Errorf("encoding broker to YAML: %w", err)
		}

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")

		_ = table.Append("Name", broker.Name)
		_ = table.Append("GUID", broker.GUID)
		_ = table.Append("URL", broker.URL)
		_ = table.Append("Created", broker.CreatedAt.Format(TimeFormatDisplay))
		_ = table.Append("Updated", broker.UpdatedAt.Format(TimeFormatDisplay))

		if broker.Relationships.Space != nil && broker.Relationships.Space.Data != nil {
			space, _ := client.Spaces().Get(ctx, broker.Relationships.Space.Data.GUID)
			if space != nil {
				_ = table.Append("Space", space.Name)
			}
		} else {
			_ = table.Append("Space", "platform (global)")
		}

		_, _ = fmt.Fprintf(os.Stdout, "Service Broker: %s\n\n", broker.Name)

		_ = table.Render()
	}

	return nil
}

func newServicesBrokersGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get BROKER_NAME_OR_GUID",
		Short: "Get service broker details",
		Long:  "Display detailed information about a specific service broker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			broker, err := findServiceBrokerByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			return renderServiceBrokerDetails(ctx, client, broker, viper.GetString("output"))
		},
	}
}

func newServicesOfferingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "offerings",
		Aliases: []string{"offering"},
		Short:   "Manage service offerings",
		Long:    "List and manage Cloud Foundry service offerings",
	}

	cmd.AddCommand(newServicesOfferingsListCommand())
	cmd.AddCommand(newServicesOfferingsGetCommand())

	return cmd
}

// renderServiceOfferingsList renders the service offerings list in the specified output format
// renderServiceOfferingsList renders the service offerings list in the specified output format.
func renderServiceOfferingsList(ctx context.Context, client capi.Client, offerings *capi.ListResponse[capi.ServiceOffering], output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(offerings.Resources)
		if err != nil {
			return fmt.Errorf("encoding offerings to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(offerings.Resources)
		if err != nil {
			return fmt.Errorf("encoding offerings to YAML: %w", err)
		}

		return nil
	default:
		return renderServiceOfferingsTable(ctx, client, offerings)
	}
}

// renderServiceOfferingsTable renders service offerings as a table
// renderServiceOfferingsTable renders service offerings as a table.
func renderServiceOfferingsTable(ctx context.Context, client capi.Client, offerings *capi.ListResponse[capi.ServiceOffering]) error {
	if len(offerings.Resources) == 0 {
		_, _ = os.Stdout.WriteString("No service offerings found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Description", "Broker", "Available", "Plans")

	for _, offering := range offerings.Resources {
		// Get broker name
		brokerName := ""

		if offering.Relationships.ServiceBroker.Data != nil {
			broker, _ := client.ServiceBrokers().Get(ctx, offering.Relationships.ServiceBroker.Data.GUID)
			if broker != nil {
				brokerName = broker.Name
			}
		}

		// Count plans
		planParams := capi.NewQueryParams()
		planParams.WithFilter("service_offering_guids", offering.GUID)
		plans, _ := client.ServicePlans().List(ctx, planParams)

		planCount := 0
		if plans != nil {
			planCount = len(plans.Resources)
		}

		available := Yes
		if !offering.Available {
			available = "no"
		}

		description := offering.Description
		if len(description) > constants.ShortDescriptionDisplayLength {
			description = description[:47] + "..."
		}

		_ = table.Append(offering.Name, description, brokerName, available, strconv.Itoa(planCount))
	}

	_ = table.Render()

	return nil
}

func newServicesOfferingsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   List,
		Short: "List service offerings",
		Long:  "List all service offerings",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := capi.NewQueryParams()

			offerings, err := client.ServiceOfferings().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list service offerings: %w", err)
			}

			return renderServiceOfferingsList(ctx, client, offerings, viper.GetString("output"))
		},
	}

	return cmd
}

// findServiceOfferingByNameOrGUID finds a service offering by name or GUID.
func findServiceOfferingByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.ServiceOffering, error) {
	// Try to get by GUID first
	offering, err := client.ServiceOfferings().Get(ctx, nameOrGUID)
	if err == nil {
		return offering, nil
	}

	// If not found by GUID, try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	offerings, err := client.ServiceOfferings().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find service offering: %w", err)
	}

	if len(offerings.Resources) == 0 {
		return nil, fmt.Errorf("service offering '%s': %w", nameOrGUID, ErrServiceOfferingNotFound)
	}

	return &offerings.Resources[0], nil
}

// renderServiceOfferingDetails renders service offering details in the specified output format.
func renderServiceOfferingDetails(offering *capi.ServiceOffering, output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(offering)
		if err != nil {
			return fmt.Errorf("encoding offering to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(offering)
		if err != nil {
			return fmt.Errorf("encoding offering to YAML: %w", err)
		}

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")

		_ = table.Append("Name", offering.Name)
		_ = table.Append("GUID", offering.GUID)
		_ = table.Append("Description", offering.Description)
		_ = table.Append("Available", strconv.FormatBool(offering.Available))
		_ = table.Append("Shareable", strconv.FormatBool(offering.Shareable))
		_ = table.Append("Created", offering.CreatedAt.Format(TimeFormatDisplay))

		if len(offering.Tags) > 0 {
			_ = table.Append("Tags", strings.Join(offering.Tags, ", "))
		}

		_, _ = fmt.Fprintf(os.Stdout, "Service Offering: %s\n\n", offering.Name)

		_ = table.Render()
	}

	return nil
}

func newServicesOfferingsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get OFFERING_NAME_OR_GUID",
		Short: "Get service offering details",
		Long:  "Display detailed information about a specific service offering",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			offering, err := findServiceOfferingByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			return renderServiceOfferingDetails(offering, viper.GetString("output"))
		},
	}
}

func newServicesPlansCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plans",
		Aliases: []string{"plan"},
		Short:   "Manage service plans",
		Long:    "List and manage Cloud Foundry service plans",
	}

	cmd.AddCommand(newServicesPlansListCommand())
	cmd.AddCommand(newServicesPlansGetCommand())
	cmd.AddCommand(newServicesPlansVisibilityCommand())

	return cmd
}

// renderServicePlansList renders the service plans list in the specified output format
// renderServicePlansList renders the service plans list in the specified output format.
func renderServicePlansList(ctx context.Context, client capi.Client, plans *capi.ListResponse[capi.ServicePlan], output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(plans.Resources)
		if err != nil {
			return fmt.Errorf("encoding plans to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(plans.Resources)
		if err != nil {
			return fmt.Errorf("encoding plans to YAML: %w", err)
		}

		return nil
	default:
		return renderServicePlansTable(ctx, client, plans)
	}
}

// renderServicePlansTable renders service plans as a table
// renderServicePlansTable renders service plans as a table.
func renderServicePlansTable(ctx context.Context, client capi.Client, plans *capi.ListResponse[capi.ServicePlan]) error {
	if len(plans.Resources) == 0 {
		_, _ = os.Stdout.WriteString("No service plans found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Service", "Description", "Free", "Available")

	for _, plan := range plans.Resources {
		// Get service offering name
		offeringName := ""

		if plan.Relationships.ServiceOffering.Data != nil {
			offering, _ := client.ServiceOfferings().Get(ctx, plan.Relationships.ServiceOffering.Data.GUID)
			if offering != nil {
				offeringName = offering.Name
			}
		}

		available := Yes
		if !plan.Available {
			available = "no"
		}

		free := Yes
		if !plan.Free {
			free = "no"
		}

		description := plan.Description
		if len(description) > constants.ShortCommandDisplayLength {
			description = description[:37] + "..."
		}

		_ = table.Append(plan.Name, offeringName, description, free, available)
	}

	_ = table.Render()

	return nil
}

func newServicesPlansListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   List,
		Short: "List service plans",
		Long:  "List all service plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := capi.NewQueryParams()

			plans, err := client.ServicePlans().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list service plans: %w", err)
			}

			return renderServicePlansList(ctx, client, plans, viper.GetString("output"))
		},
	}

	return cmd
}

// findServicePlanByNameOrGUID finds a service plan by name or GUID.
func findServicePlanByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.ServicePlan, error) {
	// Try to get by GUID first
	plan, err := client.ServicePlans().Get(ctx, nameOrGUID)
	if err == nil {
		return plan, nil
	}

	// If not found by GUID, try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	plans, err := client.ServicePlans().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find service plan: %w", err)
	}

	if len(plans.Resources) == 0 {
		return nil, fmt.Errorf("service plan '%s': %w", nameOrGUID, ErrServicePlanNotFound)
	}

	return &plans.Resources[0], nil
}

// renderServicePlanDetails renders service plan details in the specified output format.
func renderServicePlanDetails(ctx context.Context, client capi.Client, plan *capi.ServicePlan, output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(plan)
		if err != nil {
			return fmt.Errorf("encoding plan to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(plan)
		if err != nil {
			return fmt.Errorf("encoding plan to YAML: %w", err)
		}

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")

		_ = table.Append("Name", plan.Name)
		_ = table.Append("GUID", plan.GUID)
		_ = table.Append("Description", plan.Description)
		_ = table.Append("Free", strconv.FormatBool(plan.Free))
		_ = table.Append("Available", strconv.FormatBool(plan.Available))
		_ = table.Append("Visibility", plan.VisibilityType)
		_ = table.Append("Created", plan.CreatedAt.Format(TimeFormatDisplay))

		// Get service offering name
		if plan.Relationships.ServiceOffering.Data != nil {
			offering, _ := client.ServiceOfferings().Get(ctx, plan.Relationships.ServiceOffering.Data.GUID)
			if offering != nil {
				_ = table.Append("Offering", offering.Name)
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "Service Plan: %s\n\n", plan.Name)

		_ = table.Render()
	}

	return nil
}

func newServicesPlansGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get PLAN_NAME_OR_GUID",
		Short: "Get service plan details",
		Long:  "Display detailed information about a specific service plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			plan, err := findServicePlanByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			return renderServicePlanDetails(ctx, client, plan, viper.GetString("output"))
		},
	}
}

func newServicesPlansVisibilityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visibility",
		Short: "Manage service plan visibility",
		Long:  "Manage service plan visibility including getting, updating, and applying visibility settings",
	}

	cmd.AddCommand(newServicesPlansVisibilityGetCommand())
	cmd.AddCommand(newServicesPlansVisibilityUpdateCommand())
	cmd.AddCommand(newServicesPlansVisibilityApplyCommand())
	cmd.AddCommand(newServicesPlansVisibilityRemoveOrgCommand())

	return cmd
}

// renderServicePlanVisibility renders service plan visibility in the specified output format.
func renderServicePlanVisibility(plan *capi.ServicePlan, visibility *capi.ServicePlanVisibility, output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(visibility)
		if err != nil {
			return fmt.Errorf("failed to encode service plan visibility as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(visibility)
		if err != nil {
			return fmt.Errorf("failed to encode service plan visibility as YAML: %w", err)
		}

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")

		_ = table.Append("Type", visibility.Type)
		if len(visibility.Organizations) > 0 {
			var orgNames []string

			for _, org := range visibility.Organizations {
				if org.Name != "" {
					orgNames = append(orgNames, fmt.Sprintf("%s (%s)", org.Name, org.GUID))
				} else {
					orgNames = append(orgNames, org.GUID)
				}
			}

			_ = table.Append("Organizations", strings.Join(orgNames, ", "))
		}

		if visibility.Space != nil {
			spaceName := visibility.Space.GUID
			if visibility.Space.Name != "" {
				spaceName = fmt.Sprintf("%s (%s)", visibility.Space.Name, visibility.Space.GUID)
			}

			_ = table.Append("Space", spaceName)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Visibility for service plan '%s':\n\n", plan.Name)

		err := table.Render()
		if err != nil {
			return fmt.Errorf("failed to render service plan visibility table: %w", err)
		}

		return nil
	}
}

func newServicesPlansVisibilityGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get SERVICE_PLAN_NAME_OR_GUID",
		Short: "Get service plan visibility",
		Long:  "Get the current visibility settings for a service plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find service plan
			plan, err := findServicePlanByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			visibility, err := client.ServicePlans().GetVisibility(ctx, plan.GUID)
			if err != nil {
				return fmt.Errorf("getting service plan visibility: %w", err)
			}

			return renderServicePlanVisibility(plan, visibility, viper.GetString("output"))
		},
	}
}

// ServicePlanVisibilityOperation defines the operations for service plan visibility.
type ServicePlanVisibilityOperation struct {
	Use                string
	Short              string
	Long               string
	Action             string
	RequestType        func(visibilityType string, organizations []string) any
	VisibilityFunction func(context.Context, capi.ServicePlansClient, string, any) (any, error)
}

// createServicePlanVisibilityCommand creates a generic service plan visibility command.
// executeServicePlanVisibilityOperation executes a service plan visibility operation.
func executeServicePlanVisibilityOperation(
	ctx context.Context,
	client capi.Client,
	planNameOrGUID string,
	operation ServicePlanVisibilityOperation,
	visibilityType string,
	organizations []string,
) error {
	// Find service plan
	plan, err := findServicePlanByNameOrGUID(ctx, client, planNameOrGUID)
	if err != nil {
		return err
	}

	request := operation.RequestType(visibilityType, organizations)

	visibility, err := operation.VisibilityFunction(ctx, client.ServicePlans(), plan.GUID, request)
	if err != nil {
		return fmt.Errorf("%s service plan visibility: %w", operation.Action, err)
	}

	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(visibility)
		if err != nil {
			return fmt.Errorf("failed to encode visibility to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(visibility)
		if err != nil {
			return fmt.Errorf("failed to encode visibility to YAML: %w", err)
		}

		return nil
	default:
		_, _ = fmt.Fprintf(os.Stdout, "✓ Service plan '%s' visibility %s as '%s'\n", plan.Name, operation.Action, visibilityType)

		if len(organizations) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "Organizations: %s\n", strings.Join(organizations, ", "))
		}
	}

	return nil
}

func servicePlanVisibilityOrgs(guids []string) []capi.ServicePlanVisibilityOrg {
	orgs := make([]capi.ServicePlanVisibilityOrg, len(guids))
	for i, guid := range guids {
		orgs[i] = capi.ServicePlanVisibilityOrg{GUID: guid}
	}

	return orgs
}

func createServicePlanVisibilityCommand(operation ServicePlanVisibilityOperation) *cobra.Command {
	var (
		visibilityType string
		organizations  []string
	)

	cmd := &cobra.Command{
		Use:   operation.Use,
		Short: operation.Short,
		Long:  operation.Long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if visibilityType == "" {
				return ErrVisibilityTypeRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			return executeServicePlanVisibilityOperation(
				context.Background(),
				client,
				args[0],
				operation,
				visibilityType,
				organizations,
			)
		},
	}

	cmd.Flags().StringVarP(&visibilityType, "type", "t", "", "Visibility type (public, admin, organization, space)")
	cmd.Flags().StringSliceVarP(&organizations, "orgs", "o", []string{}, "Organization GUIDs (for organization visibility)")

	return cmd
}

func newServicesPlansVisibilityUpdateCommand() *cobra.Command {
	return createServicePlanVisibilityCommand(ServicePlanVisibilityOperation{
		Use:    "update SERVICE_PLAN_NAME_OR_GUID",
		Short:  "Update service plan visibility",
		Long:   "Update the visibility settings for a service plan",
		Action: "updated",
		RequestType: func(visibilityType string, organizations []string) any {
			return &capi.ServicePlanVisibilityUpdateRequest{
				Type:          visibilityType,
				Organizations: servicePlanVisibilityOrgs(organizations),
			}
		},
		VisibilityFunction: func(ctx context.Context, client capi.ServicePlansClient, guid string, request any) (any, error) {
			updateRequest, ok := request.(*capi.ServicePlanVisibilityUpdateRequest)
			if !ok {
				return nil, constants.ErrInvalidRequestType
			}

			return client.UpdateVisibility(ctx, guid, updateRequest)
		},
	})
}

func newServicesPlansVisibilityApplyCommand() *cobra.Command {
	return createServicePlanVisibilityCommand(ServicePlanVisibilityOperation{
		Use:    "apply SERVICE_PLAN_NAME_OR_GUID",
		Short:  "Apply service plan visibility",
		Long:   "Apply visibility settings to a service plan",
		Action: Applied,
		RequestType: func(visibilityType string, organizations []string) any {
			return &capi.ServicePlanVisibilityApplyRequest{
				Type:          visibilityType,
				Organizations: servicePlanVisibilityOrgs(organizations),
			}
		},
		VisibilityFunction: func(ctx context.Context, client capi.ServicePlansClient, guid string, request any) (any, error) {
			applyRequest, ok := request.(*capi.ServicePlanVisibilityApplyRequest)
			if !ok {
				return nil, constants.ErrInvalidRequestTypeForApplyVisibility
			}

			return client.ApplyVisibility(ctx, guid, applyRequest)
		},
	})
}

func newServicesPlansVisibilityRemoveOrgCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-org SERVICE_PLAN_NAME_OR_GUID ORG_GUID",
		Short: "Remove organization from service plan visibility",
		Long:  "Remove an organization from service plan visibility",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			planNameOrGUID := args[0]
			orgGUID := args[1]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find service plan
			plan, err := client.ServicePlans().Get(ctx, planNameOrGUID)
			if err != nil {
				// If not found by GUID, try by name
				params := capi.NewQueryParams()
				params.WithFilter("names", planNameOrGUID)

				plans, err := client.ServicePlans().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to find service plan: %w", err)
				}

				if len(plans.Resources) == 0 {
					return fmt.Errorf("service plan '%s': %w", planNameOrGUID, ErrServicePlanNotFound)
				}

				plan = &plans.Resources[0]
			}

			err = client.ServicePlans().RemoveOrgFromVisibility(ctx, plan.GUID, orgGUID)
			if err != nil {
				return fmt.Errorf("removing organization from service plan visibility: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "✓ Organization '%s' removed from service plan '%s' visibility\n", orgGUID, plan.Name)

			return nil
		},
	}
}
