package ag

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func dataSourceAwsSqsQueue() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSqsQueueRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsSqsQueueRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	name := d.Get("name").(string)

	urlOutput, err := conn.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(name),
	})
	if err != nil || urlOutput.QueueUrl == nil {
		return fmt.Errorf("Error getting queue URL: %w", err)
	}

	queueURL := aws.StringValue(urlOutput.QueueUrl)

	attributesOutput, err := conn.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []*string{aws.String(sqs.QueueAttributeNameQueueArn)},
	})
	if err != nil {
		return fmt.Errorf("Error getting queue attributes: %w", err)
	}

	d.Set("arn", attributesOutput.Attributes[sqs.QueueAttributeNameQueueArn])
	d.Set("url", queueURL)
	d.SetId(queueURL)

	tags, err := keyvaluetags.SqsListTags(conn, queueURL)

	if err != nil {
		return fmt.Errorf("error listing tags for SQS Queue (%s): %w", queueURL, err)
	}

	if err := d.Set("tags", tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	return nil
}
