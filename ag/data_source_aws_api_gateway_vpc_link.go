package ag

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func dataSourceAwsApiGatewayVpcLink() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsApiGatewayVpcLinkRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status_message": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_arns": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsApiGatewayVpcLinkRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigatewayconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	params := &apigateway.GetVpcLinksInput{}

	target := d.Get("name")
	var matchedVpcLinks []*apigateway.UpdateVpcLinkOutput
	log.Printf("[DEBUG] Reading API Gateway VPC links: %s", params)
	err := conn.GetVpcLinksPages(params, func(page *apigateway.GetVpcLinksOutput, lastPage bool) bool {
		for _, api := range page.Items {
			if aws.StringValue(api.Name) == target {
				matchedVpcLinks = append(matchedVpcLinks, api)
			}
		}
		return !lastPage
	})
	if err != nil {
		return fmt.Errorf("error describing API Gateway VPC links: %w", err)
	}

	if len(matchedVpcLinks) == 0 {
		return fmt.Errorf("no API Gateway VPC link with name %q found in this region", target)
	}
	if len(matchedVpcLinks) > 1 {
		return fmt.Errorf("multiple API Gateway VPC links with name %q found in this region", target)
	}

	match := matchedVpcLinks[0]

	d.SetId(aws.StringValue(match.Id))
	d.Set("name", match.Name)
	d.Set("status", match.Status)
	d.Set("status_message", match.StatusMessage)
	d.Set("description", match.Description)
	d.Set("target_arns", flattenStringList(match.TargetArns))

	if err := d.Set("tags", keyvaluetags.ApigatewayKeyValueTags(match.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	return nil
}
