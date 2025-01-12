package ag

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAwsPrefixList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsPrefixListRead,

		Schema: map[string]*schema.Schema{
			"prefix_list_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cidr_blocks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"filter": dataSourceFiltersSchema(),
		},
	}
}

func dataSourceAwsPrefixListRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	filters, filtersOk := d.GetOk("filter")

	req := &ec2.DescribePrefixListsInput{}
	if filtersOk {
		req.Filters = buildAwsDataSourceFilters(filters.(*schema.Set))
	}
	if prefixListID := d.Get("prefix_list_id"); prefixListID != "" {
		req.PrefixListIds = aws.StringSlice([]string{prefixListID.(string)})
	}
	if prefixListName := d.Get("name"); prefixListName.(string) != "" {
		req.Filters = append(req.Filters, &ec2.Filter{
			Name:   aws.String("prefix-list-name"),
			Values: aws.StringSlice([]string{prefixListName.(string)}),
		})
	}

	log.Printf("[DEBUG] Reading Prefix List: %s", req)
	resp, err := conn.DescribePrefixLists(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.PrefixLists) == 0 {
		return fmt.Errorf("no matching prefix list found; the prefix list ID or name may be invalid or not exist in the current region")
	}

	pl := resp.PrefixLists[0]

	d.SetId(aws.StringValue(pl.PrefixListId))
	d.Set("name", pl.PrefixListName)
	d.Set("cidr_blocks", aws.StringValueSlice(pl.Cidrs))

	return nil
}
