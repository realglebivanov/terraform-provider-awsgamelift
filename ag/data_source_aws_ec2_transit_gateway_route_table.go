package ag

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func dataSourceAwsEc2TransitGatewayRouteTable() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEc2TransitGatewayRouteTableRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"default_association_route_table": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"default_propagation_route_table": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"filter": dataSourceFiltersSchema(),
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"transit_gateway_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsEc2TransitGatewayRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	input := &ec2.DescribeTransitGatewayRouteTablesInput{}

	if v, ok := d.GetOk("filter"); ok {
		input.Filters = buildAwsDataSourceFilters(v.(*schema.Set))
	}

	if v, ok := d.GetOk("id"); ok {
		input.TransitGatewayRouteTableIds = []*string{aws.String(v.(string))}
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateways: %s", input)
	output, err := conn.DescribeTransitGatewayRouteTables(input)

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Route Table: %w", err)
	}

	if output == nil || len(output.TransitGatewayRouteTables) == 0 {
		return errors.New("error reading EC2 Transit Gateway Route Table: no results found")
	}

	if len(output.TransitGatewayRouteTables) > 1 {
		return errors.New("error reading EC2 Transit Gateway Route Table: multiple results found, try adjusting search criteria")
	}

	transitGatewayRouteTable := output.TransitGatewayRouteTables[0]

	if transitGatewayRouteTable == nil {
		return errors.New("error reading EC2 Transit Gateway Route Table: empty result")
	}

	d.Set("default_association_route_table", transitGatewayRouteTable.DefaultAssociationRouteTable)
	d.Set("default_propagation_route_table", transitGatewayRouteTable.DefaultPropagationRouteTable)

	if err := d.Set("tags", keyvaluetags.Ec2KeyValueTags(transitGatewayRouteTable.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	d.Set("transit_gateway_id", transitGatewayRouteTable.TransitGatewayId)

	d.SetId(aws.StringValue(transitGatewayRouteTable.TransitGatewayRouteTableId))

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   ec2.ServiceName,
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("transit-gateway-route-table/%s", d.Id()),
	}.String()

	d.Set("arn", arn)

	return nil
}
