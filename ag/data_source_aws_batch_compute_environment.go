package ag

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func dataSourceAwsBatchComputeEnvironment() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsBatchComputeEnvironmentRead,

		Schema: map[string]*schema.Schema{
			"compute_environment_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ecs_cluster_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"service_role": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),

			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status_reason": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsBatchComputeEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	params := &batch.DescribeComputeEnvironmentsInput{
		ComputeEnvironments: []*string{aws.String(d.Get("compute_environment_name").(string))},
	}
	log.Printf("[DEBUG] Reading Batch Compute Environment: %s", params)
	desc, err := conn.DescribeComputeEnvironments(params)

	if err != nil {
		return err
	}

	if len(desc.ComputeEnvironments) == 0 {
		return fmt.Errorf("no matches found for name: %s", d.Get("compute_environment_name").(string))
	}

	if len(desc.ComputeEnvironments) > 1 {
		return fmt.Errorf("multiple matches found for name: %s", d.Get("compute_environment_name").(string))
	}

	computeEnvironment := desc.ComputeEnvironments[0]
	d.SetId(aws.StringValue(computeEnvironment.ComputeEnvironmentArn))
	d.Set("arn", computeEnvironment.ComputeEnvironmentArn)
	d.Set("compute_environment_name", computeEnvironment.ComputeEnvironmentName)
	d.Set("ecs_cluster_arn", computeEnvironment.EcsClusterArn)
	d.Set("service_role", computeEnvironment.ServiceRole)
	d.Set("type", computeEnvironment.Type)
	d.Set("status", computeEnvironment.Status)
	d.Set("status_reason", computeEnvironment.StatusReason)
	d.Set("state", computeEnvironment.State)

	if err := d.Set("tags", keyvaluetags.BatchKeyValueTags(computeEnvironment.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	return nil
}
