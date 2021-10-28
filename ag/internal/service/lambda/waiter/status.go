package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/lambda/finder"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/tfresource"
)

const (
	EventSourceMappingStateCreating  = "Creating"
	EventSourceMappingStateDeleting  = "Deleting"
	EventSourceMappingStateDisabled  = "Disabled"
	EventSourceMappingStateDisabling = "Disabling"
	EventSourceMappingStateEnabled   = "Enabled"
	EventSourceMappingStateEnabling  = "Enabling"
	EventSourceMappingStateUpdating  = "Updating"
)

func EventSourceMappingState(conn *lambda.Lambda, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		eventSourceMappingConfiguration, err := finder.EventSourceMappingConfigurationByID(conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return eventSourceMappingConfiguration, aws.StringValue(eventSourceMappingConfiguration.State), nil
	}
}
