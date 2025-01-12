package ag

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func dataSourceAwsNetworkInterfaces() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsNetworkInterfacesRead,
		Schema: map[string]*schema.Schema{

			"filter": ec2CustomFiltersSchema(),

			"tags": tagsSchemaComputed(),

			"ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceAwsNetworkInterfacesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeNetworkInterfacesInput{}

	filters, filtersOk := d.GetOk("filter")
	tags, tagsOk := d.GetOk("tags")

	if tagsOk {
		req.Filters = buildEC2TagFilterList(
			keyvaluetags.New(tags.(map[string]interface{})).Ec2Tags(),
		)
	}

	if filtersOk {
		req.Filters = append(req.Filters, buildEC2CustomFilterList(
			filters.(*schema.Set),
		)...)
	}

	if len(req.Filters) == 0 {
		req.Filters = nil
	}

	log.Printf("[DEBUG] DescribeNetworkInterfaces %s\n", req)
	resp, err := conn.DescribeNetworkInterfaces(req)
	if err != nil {
		return err
	}

	if resp == nil || len(resp.NetworkInterfaces) == 0 {
		return errors.New("no matching network interfaces found")
	}

	networkInterfaces := make([]string, 0)

	for _, networkInterface := range resp.NetworkInterfaces {
		networkInterfaces = append(networkInterfaces, aws.StringValue(networkInterface.NetworkInterfaceId))
	}

	d.SetId(meta.(*AWSClient).region)

	if err := d.Set("ids", networkInterfaces); err != nil {
		return fmt.Errorf("Error setting network interfaces ids: %w", err)
	}

	return nil
}
