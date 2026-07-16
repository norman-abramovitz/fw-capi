package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRoutePoliciesCommand creates the route-policies command group
// (CF v3 3.225.0, experimental).
func NewRoutePoliciesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "route-policies",
		Aliases: []string{"route-policy"},
		Short:   "Manage route policies",
		Long:    "List and manage route policies on identity-aware domains (experimental)",
	}

	cmd.AddCommand(newRoutePoliciesListCommand())
	cmd.AddCommand(newRoutePoliciesCreateCommand())
	cmd.AddCommand(newRoutePoliciesGetCommand())
	cmd.AddCommand(newRoutePoliciesUpdateCommand())
	cmd.AddCommand(newRoutePoliciesDeleteCommand())

	return cmd
}

func newRoutePoliciesListCommand() *cobra.Command {
	var (
		routeGUID  string
		sourceGUID string
		source     string
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List route policies",
		Long:  "List all route policies the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			var opts []capi.RoutePolicyListOption

			if routeGUID != "" {
				opts = append(opts, capi.WithRoutePolicyRouteGUIDs(routeGUID))
			}

			if sourceGUID != "" {
				opts = append(opts, capi.WithRoutePolicySourceGUIDs(sourceGUID))
			}

			if source != "" {
				opts = append(opts, capi.WithRoutePolicySources(source))
			}

			policies, err := client.RoutePolicies().List(ctx, nil, opts...)
			if err != nil {
				return fmt.Errorf("failed to list route policies: %w", err)
			}

			return renderRoutePoliciesOutput(policies.Resources)
		},
	}

	cmd.Flags().StringVar(&routeGUID, "route", "", "filter by route GUID")
	cmd.Flags().StringVar(&sourceGUID, "source-guid", "", "filter by source GUID (app, space, or org)")
	cmd.Flags().StringVar(&source, "source", "", "filter by exact source (e.g. cf:any, cf:app:GUID)")

	return cmd
}

func renderRoutePoliciesOutput(policies []capi.RoutePolicy) error {
	renderer := &StandardOutputRenderer[capi.RoutePolicy]{
		RenderTable: func(resources []capi.RoutePolicy, pag *capi.Pagination, allPgs bool) error {
			return renderRoutePoliciesTable(resources)
		},
	}

	output := viper.GetString("output")

	return renderer.Render(policies, nil, false, output)
}

func renderRoutePoliciesTable(policies []capi.RoutePolicy) error {
	if len(policies) == 0 {
		_, _ = os.Stdout.WriteString("No route policies found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "Source", "Route GUID", "Created", "Updated")

	for _, policy := range policies {
		routeGUID := ""
		if policy.Relationships.Route.Data != nil {
			routeGUID = policy.Relationships.Route.Data.GUID
		}

		_ = table.Append(policy.GUID, policy.Source, routeGUID,
			policy.CreatedAt.Format(TimeFormatDisplay),
			policy.UpdatedAt.Format(TimeFormatDisplay))
	}

	_ = table.Render()

	return nil
}

func newRoutePoliciesCreateCommand() *cobra.Command {
	var (
		source    string
		routeGUID string
		labels    map[string]string
	)

	cmd := &cobra.Command{
		Use:   Create,
		Short: "Create a route policy",
		Long: "Create a route policy for a route on an identity-aware domain. " +
			"The source must be cf:app:GUID, cf:space:GUID, cf:org:GUID, or cf:any.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			createReq := &capi.RoutePolicyCreateRequest{
				Source: source,
				Relationships: capi.RoutePolicyRelationships{
					Route: capi.Relationship{
						Data: &capi.RelationshipData{GUID: routeGUID},
					},
				},
			}

			if labels != nil {
				createReq.Metadata = &capi.Metadata{
					Labels: labels,
				}
			}

			policy, err := client.RoutePolicies().Create(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create route policy: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully created route policy '%s' (source %s)\n", policy.GUID, policy.Source)

			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "policy source selector (required)")
	cmd.Flags().StringVar(&routeGUID, "route", "", "route GUID the policy applies to (required)")
	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("route")

	return cmd
}

func newRoutePoliciesGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get POLICY_GUID",
		Short: "Get route policy details",
		Long:  "Display detailed information about a specific route policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			policy, err := client.RoutePolicies().Get(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get route policy: %w", err)
			}

			return renderRoutePoliciesOutput([]capi.RoutePolicy{*policy})
		},
	}
}

func newRoutePoliciesUpdateCommand() *cobra.Command {
	var labels map[string]string

	cmd := &cobra.Command{
		Use:   "update POLICY_GUID",
		Short: "Update a route policy",
		Long:  "Update route policy metadata (source and route are immutable)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			updateReq := &capi.RoutePolicyUpdateRequest{}

			if labels != nil {
				updateReq.Metadata = &capi.Metadata{
					Labels: labels,
				}
			}

			policy, err := client.RoutePolicies().Update(context.Background(), args[0], updateReq)
			if err != nil {
				return fmt.Errorf("failed to update route policy: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated route policy '%s'\n", policy.GUID)

			return nil
		},
	}

	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")

	return cmd
}

func newRoutePoliciesDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete POLICY_GUID",
		Short: "Delete a route policy",
		Long:  "Delete a route policy, removing access for that source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			err = client.RoutePolicies().Delete(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("failed to delete route policy: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully deleted route policy '%s'\n", args[0])

			return nil
		},
	}
}
