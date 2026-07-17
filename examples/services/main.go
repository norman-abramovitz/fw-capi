package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/fivetwenty-io/capi/v3/pkg/cfclient"
)

const (
	// serviceInstanceTypeManaged is the CF API discriminator for a brokered service instance.
	serviceInstanceTypeManaged = "managed"
	// tagManaged marks an example service instance's Tags as backed by a managed service.
	tagManaged = "managed"
	// tagUpdated marks an example service instance's Tags as having been updated.
	tagUpdated = "updated"
	// labelKeyUpdated is the metadata label key recording that a resource was updated.
	labelKeyUpdated = "updated"
)

func main() {
	// Create authenticated client
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		log.Fatalf("Failed to create CF client: %v", err)
	}

	// Example 1: List Service Offerings and Plans
	log.Println("=== Service Offerings and Plans ===")
	listServiceOfferingsAndPlans(client, ctx)

	// Example 2: Create Managed Service Instance
	log.Println("\n=== Create Managed Service Instance ===")

	managedInstance := createManagedServiceInstance(client, ctx)

	// Example 3: Create User-Provided Service Instance
	log.Println("\n=== Create User-Provided Service Instance ===")

	userProvidedInstance := createUserProvidedServiceInstance(client, ctx)

	// Example 4: Service Instance Management
	log.Println("\n=== Service Instance Management ===")

	if managedInstance != nil {
		manageServiceInstance(client, ctx, managedInstance)
	}

	// Example 5: Service Bindings
	log.Println("\n=== Service Bindings ===")

	if managedInstance != nil {
		manageServiceBindings(client, ctx, managedInstance)
	}

	// Example 6: Service Keys
	log.Println("\n=== Service Keys ===")

	if userProvidedInstance != nil {
		manageServiceKeys(client, ctx, userProvidedInstance)
	}

	// Example 7: Service Usage Events
	log.Println("\n=== Service Usage Events ===")
	listServiceUsageEvents(client, ctx)

	// Cleanup
	log.Println("\n=== Cleanup ===")
	cleanup(client, ctx, managedInstance, userProvidedInstance)
}

func listServiceOfferingsAndPlans(client capi.Client, ctx context.Context) {
	// List all service offerings
	offerings, err := client.ServiceOfferings().List(ctx, nil)
	if err != nil {
		log.Printf("Failed to list service offerings: %v", err)

		return
	}

	log.Printf("Found %d service offerings:\n", len(offerings.Resources))

	for _, offering := range offerings.Resources {
		log.Printf("  📦 %s (%s)\n", offering.Name, offering.GUID)
		log.Printf("     Description: %s\n", offering.Description)
		log.Printf("     Broker GUID: %s\n", offering.Relationships.ServiceBroker.Data.GUID)

		if offering.Metadata != nil && len(offering.Metadata.Labels) > 0 {
			log.Println("     Labels:")

			for key, value := range offering.Metadata.Labels {
				log.Printf("       %s: %s\n", key, value)
			}
		}

		// List plans for this offering
		params := capi.NewQueryParams().WithFilter("service_offering_guids", offering.GUID)

		plans, err := client.ServicePlans().List(ctx, params)
		if err != nil {
			log.Printf("Failed to list plans for offering %s: %v", offering.Name, err)

			continue
		}

		log.Printf("     Plans (%d):\n", len(plans.Resources))

		for _, plan := range plans.Resources {
			log.Printf("       • %s (%s)\n", plan.Name, plan.GUID)
			log.Printf("         Description: %s\n", plan.Description)
			log.Printf("         Free: %v\n", plan.Free)
			log.Printf("         Available: %v\n", plan.Available)

			if len(plan.Costs) > 0 {
				log.Println("         Costs:")

				for _, cost := range plan.Costs {
					log.Printf("           %s: %.2f %s\n", cost.Unit, cost.Amount, cost.Currency)
				}
			}
		}

		log.Println()
	}
}

func findAvailableServicePlan(ctx context.Context, client capi.Client) (*capi.ServiceOffering, *capi.ServicePlan, error) {
	// Find a service plan to use (we'll use the first available plan)
	offerings, err := client.ServiceOfferings().List(ctx, nil)
	if err != nil || len(offerings.Resources) == 0 {
		return nil, nil, fmt.Errorf("no service offerings available: %w", err)
	}

	offering := offerings.Resources[0]
	params := capi.NewQueryParams().WithFilter("service_offering_guids", offering.GUID)

	plans, err := client.ServicePlans().List(ctx, params)
	if err != nil || len(plans.Resources) == 0 {
		return nil, nil, fmt.Errorf("no service plans available for offering %s: %w", offering.Name, err)
	}

	return &offering, &plans.Resources[0], nil
}

func buildManagedServiceInstanceRequest(spaceGUID, planGUID string) *capi.ServiceInstanceCreateRequest {
	return &capi.ServiceInstanceCreateRequest{
		Type: serviceInstanceTypeManaged,
		Name: "example-managed-service",
		Relationships: capi.ServiceInstanceRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{GUID: spaceGUID},
			},
			ServicePlan: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: planGUID},
			},
		},
		Parameters: map[string]any{
			"example_param": "example_value",
		},
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"team":        "platform",
				"environment": "demo",
			},
			Annotations: map[string]string{
				"created-by": "service-example",
			},
		},
		Tags: []string{"database", tagManaged},
	}
}

func printServiceInstanceInfo(instance *capi.ServiceInstance, planName string) {
	log.Printf("Created managed service instance: %s (GUID: %s)\n", instance.Name, instance.GUID)
	log.Printf("  Type: %s\n", instance.Type)
	log.Printf("  State: %s\n", instance.LastOperation.State)
	log.Printf("  Service Plan: %s\n", planName)
}

func createManagedServiceInstance(client capi.Client, ctx context.Context) *capi.ServiceInstance {
	// Get a space to work with (you'll need to replace with your actual space GUID)
	spaceGUID := "your-space-guid"

	// Find an available service plan
	_, plan, err := findAvailableServicePlan(ctx, client)
	if err != nil {
		log.Printf("Failed to find service plan: %v", err)

		return nil
	}

	// Build the service instance creation request
	createReq := buildManagedServiceInstanceRequest(spaceGUID, plan.GUID)

	// Create the managed service instance
	instanceInterface, err := client.ServiceInstances().Create(ctx, createReq)
	if err != nil {
		log.Printf("Failed to create managed service instance: %v", err)

		return nil
	}

	instance, ok := instanceInterface.(*capi.ServiceInstance)
	if !ok {
		log.Printf("Unexpected return type from ServiceInstances().Create()")

		return nil
	}

	// Print service instance information
	printServiceInstanceInfo(instance, plan.Name)

	// Wait for the service instance to be ready
	if instance.LastOperation.State == "in progress" {
		log.Println("Waiting for service instance to be ready...")

		instance = waitForServiceInstanceReady(client, ctx, instance.GUID)
	}

	return instance
}

func createUserProvidedServiceInstance(client capi.Client, ctx context.Context) *capi.ServiceInstance {
	// Get a space to work with
	spaceGUID := "your-space-guid"

	// Create user-provided service instance
	createReq := &capi.ServiceInstanceCreateRequest{
		Type: "user-provided",
		Name: "example-user-provided-service",
		Relationships: capi.ServiceInstanceRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{GUID: spaceGUID},
			},
		},
		Credentials: map[string]any{
			"uri":      "https://external-api.example.com",
			"api_key":  "secret-api-key",
			"username": "service-user",
			"password": "service-password",
		},
		SyslogDrainURL:  &[]string{"https://logs.example.com/drain"}[0],
		RouteServiceURL: &[]string{"https://route-service.example.com"}[0],
		Tags:            []string{"external", "api"},
	}

	instanceInterface, err := client.ServiceInstances().Create(ctx, createReq)
	if err != nil {
		log.Printf("Failed to create user-provided service instance: %v", err)

		return nil
	}

	instance, ok := instanceInterface.(*capi.ServiceInstance)
	if !ok {
		log.Printf("Unexpected return type from ServiceInstances().Create()")

		return nil
	}

	log.Printf("Created user-provided service instance: %s (GUID: %s)\n", instance.Name, instance.GUID)
	log.Printf("  Type: %s\n", instance.Type)
	log.Printf("  Syslog Drain URL: %s\n", *instance.SyslogDrainURL)
	log.Printf("  Route Service URL: %s\n", *instance.RouteServiceURL)

	return instance
}

func manageServiceInstance(client capi.Client, ctx context.Context, instance *capi.ServiceInstance) {
	log.Printf("Managing service instance: %s\n", instance.Name)

	// Get updated service instance details
	updatedInstance, err := client.ServiceInstances().Get(ctx, instance.GUID)
	if err != nil {
		log.Printf("Failed to get service instance: %v", err)

		return
	}

	log.Printf("Service Instance Details:\n")
	log.Printf("  Name: %s\n", updatedInstance.Name)
	log.Printf("  Type: %s\n", updatedInstance.Type)
	log.Printf("  State: %s\n", updatedInstance.LastOperation.State)
	log.Printf("  Created: %s\n", updatedInstance.CreatedAt.Format(time.RFC3339))
	log.Printf("  Updated: %s\n", updatedInstance.UpdatedAt.Format(time.RFC3339))

	if updatedInstance.Tags != nil {
		log.Printf("  Tags: %v\n", updatedInstance.Tags)
	}

	// Update service instance
	newName := "updated-service-name"
	updateReq := &capi.ServiceInstanceUpdateRequest{
		Name: &newName,
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"version":       "2.0",
				labelKeyUpdated: "true",
			},
		},
		Tags: []string{"database", tagManaged, tagUpdated},
	}

	updatedInstanceInterface, err := client.ServiceInstances().Update(ctx, instance.GUID, updateReq)
	if err != nil {
		log.Printf("Failed to update service instance: %v", err)

		return
	}

	updatedInstance, ok := updatedInstanceInterface.(*capi.ServiceInstance)
	if !ok {
		log.Printf("Unexpected return type from ServiceInstances().Update()")

		return
	}

	log.Printf("Updated service instance name to: %s\n", updatedInstance.Name)

	// Get service instance parameters (if available)
	parameters, err := client.ServiceInstances().GetParameters(ctx, instance.GUID)
	if err != nil {
		log.Printf("Failed to get parameters: %v", err)
	} else {
		log.Printf("Service parameters available: %v\n", len(parameters.Parameters) > 0)
	}
}

func createServiceBinding(ctx context.Context, client capi.Client, serviceInstanceGUID, appGUID string) (*capi.ServiceCredentialBinding, error) {
	bindingName := "example-binding"
	createBindingReq := &capi.ServiceCredentialBindingCreateRequest{
		Type: "app",
		Name: &bindingName,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{GUID: serviceInstanceGUID},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: appGUID},
			},
		},
		Parameters: map[string]any{
			"permission": "read-write",
			"pool_size":  constants.DefaultPageSize,
		},
	}

	bindingInterface, err := client.ServiceCredentialBindings().Create(ctx, createBindingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create service binding: %w", err)
	}

	binding, ok := bindingInterface.(*capi.ServiceCredentialBinding)
	if !ok {
		return nil, constants.ErrUnexpectedServiceCredentialBinding
	}

	log.Printf("Created service binding: %s (GUID: %s)\n", binding.Name, binding.GUID)

	return binding, nil
}

func listServiceBindings(ctx context.Context, client capi.Client, serviceInstanceGUID string) error {
	params := capi.NewQueryParams().WithFilter("service_instance_guids", serviceInstanceGUID)

	bindings, err := client.ServiceCredentialBindings().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list service bindings: %w", err)
	}

	log.Printf("Service instance has %d bindings:\n", len(bindings.Resources))

	for _, b := range bindings.Resources {
		log.Printf("  - %s (Type: %s, State: %s)\n", b.Name, b.Type, b.LastOperation.State)
	}

	return nil
}

func showBindingDetails(ctx context.Context, client capi.Client, bindingGUID string) error {
	bindingDetails, err := client.ServiceCredentialBindings().GetDetails(ctx, bindingGUID)
	if err != nil {
		return fmt.Errorf("failed to get binding details: %w", err)
	}

	log.Printf("Binding has %d credential keys\n", len(bindingDetails.Credentials))

	return nil
}

func updateServiceBinding(ctx context.Context, client capi.Client, bindingGUID string) error {
	updateBindingReq := &capi.ServiceCredentialBindingUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				labelKeyUpdated: "true",
			},
		},
	}

	updatedBinding, err := client.ServiceCredentialBindings().Update(ctx, bindingGUID, updateBindingReq)
	if err != nil {
		return fmt.Errorf("failed to update binding: %w", err)
	}

	log.Printf("Updated binding name to: %s\n", updatedBinding.Name)

	return nil
}

func deleteServiceBinding(ctx context.Context, client capi.Client, binding *capi.ServiceCredentialBinding) error {
	_, err := client.ServiceCredentialBindings().Delete(ctx, binding.GUID)
	if err != nil {
		return fmt.Errorf("failed to delete service binding: %w", err)
	}

	log.Printf("Deleted service binding: %s\n", binding.Name)

	return nil
}

func manageServiceBindings(client capi.Client, ctx context.Context, serviceInstance *capi.ServiceInstance) {
	// Get an application to bind to (you'll need to replace with actual app GUID)
	appGUID := "your-app-guid"

	log.Printf("Managing service bindings for: %s\n", serviceInstance.Name)

	// Create service credential binding (app binding)
	binding, err := createServiceBinding(ctx, client, serviceInstance.GUID, appGUID)
	if err != nil {
		log.Printf("%v", err)

		return
	}

	// Wait for binding to be ready
	if binding.LastOperation != nil && binding.LastOperation.State == "in progress" {
		log.Println("Waiting for service binding to be ready...")

		binding = waitForServiceBindingReady(client, ctx, binding.GUID)
	}

	// List all bindings for the service instance
	err = listServiceBindings(ctx, client, serviceInstance.GUID)
	if err != nil {
		log.Printf("%v", err)

		return
	}

	// Get binding details
	err = showBindingDetails(ctx, client, binding.GUID)
	if err != nil {
		log.Printf("%v", err)
	}

	// Update binding
	err = updateServiceBinding(ctx, client, binding.GUID)
	if err != nil {
		log.Printf("%v", err)
	}

	// Cleanup: Delete the binding
	err = deleteServiceBinding(ctx, client, binding)
	if err != nil {
		log.Printf("%v", err)
	}
}

func createServiceKey(ctx context.Context, client capi.Client, serviceInstanceGUID string) (*capi.ServiceCredentialBinding, error) {
	keyName := "example-service-key"
	createKeyReq := &capi.ServiceCredentialBindingCreateRequest{
		Type: "key",
		Name: &keyName,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{GUID: serviceInstanceGUID},
			},
		},
		Parameters: map[string]any{
			"permissions": []string{"read", "write"},
		},
	}

	serviceKeyInterface, err := client.ServiceCredentialBindings().Create(ctx, createKeyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create service key: %w", err)
	}

	serviceKey, ok := serviceKeyInterface.(*capi.ServiceCredentialBinding)
	if !ok {
		return nil, constants.ErrUnexpectedServiceCredentialBinding
	}

	log.Printf("Created service key: %s (GUID: %s)\n", serviceKey.Name, serviceKey.GUID)

	return serviceKey, nil
}

func listServiceKeys(ctx context.Context, client capi.Client, serviceInstanceGUID string) error {
	params := capi.NewQueryParams()
	params.WithFilter("service_instance_guids", serviceInstanceGUID)
	params.WithFilter("type", "key")

	serviceKeys, err := client.ServiceCredentialBindings().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list service keys: %w", err)
	}

	log.Printf("Service instance has %d keys:\n", len(serviceKeys.Resources))

	for _, key := range serviceKeys.Resources {
		log.Printf("  - %s (GUID: %s)\n", key.Name, key.GUID)
	}

	return nil
}

func showServiceKeyCredentials(ctx context.Context, client capi.Client, serviceKeyGUID string) error {
	keyDetails, err := client.ServiceCredentialBindings().GetDetails(ctx, serviceKeyGUID)
	if err != nil {
		return fmt.Errorf("failed to get service key details: %w", err)
	}

	log.Printf("Service key credentials available\n")

	for credKey := range keyDetails.Credentials {
		log.Printf("  - %s\n", credKey)
	}

	return nil
}

func deleteServiceKey(ctx context.Context, client capi.Client, serviceKey *capi.ServiceCredentialBinding) error {
	_, err := client.ServiceCredentialBindings().Delete(ctx, serviceKey.GUID)
	if err != nil {
		return fmt.Errorf("failed to delete service key: %w", err)
	}

	log.Printf("Deleted service key: %s\n", serviceKey.Name)

	return nil
}

func manageServiceKeys(client capi.Client, ctx context.Context, serviceInstance *capi.ServiceInstance) {
	log.Printf("Managing service keys for: %s\n", serviceInstance.Name)

	// Create service key (credential binding without app)
	serviceKey, err := createServiceKey(ctx, client, serviceInstance.GUID)
	if err != nil {
		log.Printf("%v", err)

		return
	}

	// List service keys for the instance
	err = listServiceKeys(ctx, client, serviceInstance.GUID)
	if err != nil {
		log.Printf("%v", err)

		return
	}

	// Get service key credentials
	err = showServiceKeyCredentials(ctx, client, serviceKey.GUID)
	if err != nil {
		log.Printf("%v", err)
	}

	// Cleanup: Delete the service key
	err = deleteServiceKey(ctx, client, serviceKey)
	if err != nil {
		log.Printf("%v", err)
	}
}

func listServiceUsageEvents(client capi.Client, ctx context.Context) {
	log.Println("=== Service Usage Events ===")
	log.Println("Service usage events are not implemented in this client yet")
}

func cleanup(client capi.Client, ctx context.Context, managedInstance, userProvidedInstance *capi.ServiceInstance) {
	if managedInstance != nil {
		log.Printf("Deleting managed service instance: %s\n", managedInstance.Name)

		_, err := client.ServiceInstances().Delete(ctx, managedInstance.GUID)
		if err != nil {
			log.Printf("Failed to delete managed service instance: %v", err)
		} else {
			log.Printf("Initiated deletion of managed service instance\n")
		}
	}

	if userProvidedInstance != nil {
		log.Printf("Deleting user-provided service instance: %s\n", userProvidedInstance.Name)

		_, err := client.ServiceInstances().Delete(ctx, userProvidedInstance.GUID)
		if err != nil {
			log.Printf("Failed to delete user-provided service instance: %v", err)
		} else {
			log.Printf("Deleted user-provided service instance\n")
		}
	}
}

func waitForServiceInstanceReady(client capi.Client, ctx context.Context, instanceGUID string) *capi.ServiceInstance {
	maxAttempts := 30
	for range maxAttempts {
		instance, err := client.ServiceInstances().Get(ctx, instanceGUID)
		if err != nil {
			log.Printf("Error checking service instance status: %v", err)

			return nil
		}

		switch instance.LastOperation.State {
		case "succeeded":
			log.Printf("Service instance is ready!\n")

			return instance
		case "failed":
			log.Printf("Service instance creation failed: %s\n", instance.LastOperation.Description)

			return instance
		}

		log.Printf("Service instance status: %s (%s)\n",
			instance.LastOperation.State, instance.LastOperation.Description)
		time.Sleep(constants.VeryLongPollInterval)
	}

	log.Printf("Timeout waiting for service instance to be ready\n")

	return nil
}

func waitForServiceBindingReady(client capi.Client, ctx context.Context, bindingGUID string) *capi.ServiceCredentialBinding {
	maxAttempts := 20
	for range maxAttempts {
		binding, err := client.ServiceCredentialBindings().Get(ctx, bindingGUID)
		if err != nil {
			log.Printf("Error checking service binding status: %v", err)

			return nil
		}

		switch binding.LastOperation.State {
		case "succeeded":
			log.Printf("Service binding is ready!\n")

			return binding
		case "failed":
			description := ""
			if binding.LastOperation.Description != nil {
				description = *binding.LastOperation.Description
			}

			log.Printf("Service binding creation failed: %s\n", description)

			return binding
		}

		log.Printf("Service binding status: %s\n", binding.LastOperation.State)
		time.Sleep(constants.LongPollInterval)
	}

	log.Printf("Timeout waiting for service binding to be ready\n")

	return nil
}
