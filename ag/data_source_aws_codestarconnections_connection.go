package ag

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/codestarconnections/finder"
)

func dataSourceAwsCodeStarConnectionsConnection() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCodeStarConnectionsConnectionRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"connection_status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"host_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"provider_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsCodeStarConnectionsConnectionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codestarconnectionsconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	arn := d.Get("arn").(string)

	log.Printf("[DEBUG] Getting CodeStar Connection")
	connection, err := finder.ConnectionByArn(conn, arn)
	if err != nil {
		return fmt.Errorf("error getting CodeStar Connection (%s): %w", arn, err)
	}
	log.Printf("[DEBUG] CodeStar Connection: %#v", connection)

	d.SetId(arn)
	d.Set("connection_status", connection.ConnectionStatus)
	d.Set("host_arn", connection.HostArn)
	d.Set("name", connection.ConnectionName)
	d.Set("provider_type", connection.ProviderType)

	tags, err := keyvaluetags.CodestarconnectionsListTags(conn, arn)
	if err != nil {
		return fmt.Errorf("error listing tags for CodeStar Connection (%s): %w", arn, err)
	}

	if err := d.Set("tags", tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags for CodeStar Connection (%s): %w", arn, err)
	}

	return nil
}
