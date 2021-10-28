package waiter

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/sqs/finder"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/tfresource"
)

func QueueState(conn *sqs.SQS, url string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.QueueAttributesByURL(conn, url)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, queueStateExists, nil
	}
}
