package ag

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func dataSourceAwsCloudwatchLogGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCloudwatchLogGroupRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"creation_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"retention_in_days": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsCloudwatchLogGroupRead(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	conn := meta.(*AWSClient).cloudwatchlogsconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	logGroup, err := lookupCloudWatchLogGroup(conn, name)
	if err != nil {
		return err
	}
	if logGroup == nil {
		return fmt.Errorf("No log group named %s found\n", name)
	}

	d.SetId(name)
	d.Set("arn", logGroup.Arn)
	d.Set("creation_time", logGroup.CreationTime)
	d.Set("retention_in_days", logGroup.RetentionInDays)
	d.Set("kms_key_id", logGroup.KmsKeyId)

	tags, err := keyvaluetags.CloudwatchlogsListTags(conn, name)

	if err != nil {
		return fmt.Errorf("error listing tags for CloudWatch Logs Group (%s): %w", name, err)
	}

	if err := d.Set("tags", tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	return nil
}
