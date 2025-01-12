package ag

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsEc2TransitGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TransitGatewayCreate,
		Read:   resourceAwsEc2TransitGatewayRead,
		Update: resourceAwsEc2TransitGatewayUpdate,
		Delete: resourceAwsEc2TransitGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		CustomizeDiff: customdiff.Sequence(
			customdiff.ForceNewIfChange("default_route_table_association", func(_ context.Context, old, new, meta interface{}) bool {
				// Only changes from disable to enable for feature_set should force a new resource
				return old.(string) == ec2.DefaultRouteTableAssociationValueDisable && new.(string) == ec2.DefaultRouteTableAssociationValueEnable
			}),
			customdiff.ForceNewIfChange("default_route_table_propagation", func(_ context.Context, old, new, meta interface{}) bool {
				// Only changes from disable to enable for feature_set should force a new resource
				return old.(string) == ec2.DefaultRouteTablePropagationValueDisable && new.(string) == ec2.DefaultRouteTablePropagationValueEnable
			}),
			SetTagsDiff,
		),

		Schema: map[string]*schema.Schema{
			"amazon_side_asn": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  64512,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"association_default_route_table_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_accept_shared_attachments": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.AutoAcceptSharedAttachmentsValueDisable,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.AutoAcceptSharedAttachmentsValueDisable,
					ec2.AutoAcceptSharedAttachmentsValueEnable,
				}, false),
			},
			"default_route_table_association": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.DefaultRouteTableAssociationValueEnable,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.DefaultRouteTableAssociationValueDisable,
					ec2.DefaultRouteTableAssociationValueEnable,
				}, false),
			},
			"default_route_table_propagation": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.DefaultRouteTablePropagationValueEnable,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.DefaultRouteTablePropagationValueDisable,
					ec2.DefaultRouteTablePropagationValueEnable,
				}, false),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dns_support": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.DnsSupportValueEnable,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.DnsSupportValueDisable,
					ec2.DnsSupportValueEnable,
				}, false),
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"propagation_default_route_table_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
			"vpn_ecmp_support": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.VpnEcmpSupportValueEnable,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.VpnEcmpSupportValueDisable,
					ec2.VpnEcmpSupportValueEnable,
				}, false),
			},
		},
	}
}

func resourceAwsEc2TransitGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	input := &ec2.CreateTransitGatewayInput{
		Options: &ec2.TransitGatewayRequestOptions{
			AutoAcceptSharedAttachments:  aws.String(d.Get("auto_accept_shared_attachments").(string)),
			DefaultRouteTableAssociation: aws.String(d.Get("default_route_table_association").(string)),
			DefaultRouteTablePropagation: aws.String(d.Get("default_route_table_propagation").(string)),
			DnsSupport:                   aws.String(d.Get("dns_support").(string)),
			VpnEcmpSupport:               aws.String(d.Get("vpn_ecmp_support").(string)),
		},
		TagSpecifications: ec2TagSpecificationsFromKeyValueTags(tags, ec2.ResourceTypeTransitGateway),
	}

	if v, ok := d.GetOk("amazon_side_asn"); ok {
		input.Options.AmazonSideAsn = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating EC2 Transit Gateway: %s", input)
	output, err := conn.CreateTransitGateway(input)
	if err != nil {
		return fmt.Errorf("error creating EC2 Transit Gateway: %s", err)
	}

	d.SetId(aws.StringValue(output.TransitGateway.TransitGatewayId))

	if err := waitForEc2TransitGatewayCreation(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway (%s) availability: %s", d.Id(), err)
	}

	return resourceAwsEc2TransitGatewayRead(d, meta)
}

func resourceAwsEc2TransitGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	transitGateway, err := ec2DescribeTransitGateway(conn, d.Id())

	if isAWSErr(err, "InvalidTransitGatewayID.NotFound", "") {
		log.Printf("[WARN] EC2 Transit Gateway (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway: %s", err)
	}

	if transitGateway == nil {
		log.Printf("[WARN] EC2 Transit Gateway (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if aws.StringValue(transitGateway.State) == ec2.TransitGatewayStateDeleting || aws.StringValue(transitGateway.State) == ec2.TransitGatewayStateDeleted {
		log.Printf("[WARN] EC2 Transit Gateway (%s) in deleted state (%s), removing from state", d.Id(), aws.StringValue(transitGateway.State))
		d.SetId("")
		return nil
	}

	if transitGateway.Options == nil {
		return fmt.Errorf("error reading EC2 Transit Gateway (%s): missing options", d.Id())
	}

	d.Set("amazon_side_asn", transitGateway.Options.AmazonSideAsn)
	d.Set("arn", transitGateway.TransitGatewayArn)
	d.Set("association_default_route_table_id", transitGateway.Options.AssociationDefaultRouteTableId)
	d.Set("auto_accept_shared_attachments", transitGateway.Options.AutoAcceptSharedAttachments)
	d.Set("default_route_table_association", transitGateway.Options.DefaultRouteTableAssociation)
	d.Set("default_route_table_propagation", transitGateway.Options.DefaultRouteTablePropagation)
	d.Set("description", transitGateway.Description)
	d.Set("dns_support", transitGateway.Options.DnsSupport)
	d.Set("owner_id", transitGateway.OwnerId)
	d.Set("propagation_default_route_table_id", transitGateway.Options.PropagationDefaultRouteTableId)

	tags := keyvaluetags.Ec2KeyValueTags(transitGateway.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	d.Set("vpn_ecmp_support", transitGateway.Options.VpnEcmpSupport)

	return nil
}

func resourceAwsEc2TransitGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	modifyTransitGatewayInput := &ec2.ModifyTransitGatewayInput{}
	transitGatewayModified := false

	if d.HasChange("description") {
		transitGatewayModified = true
		modifyTransitGatewayInput.Description = aws.String(d.Get("description").(string))
	}

	options := &ec2.ModifyTransitGatewayOptions{}

	if d.HasChange("auto_accept_shared_attachments") {
		transitGatewayModified = true
		options.AutoAcceptSharedAttachments = aws.String(d.Get("auto_accept_shared_attachments").(string))
	}

	if d.HasChange("default_route_table_association") {
		transitGatewayModified = true
		options.DefaultRouteTableAssociation = aws.String(d.Get("default_route_table_association").(string))
	}

	if d.HasChange("default_route_table_propagation") {
		transitGatewayModified = true
		options.DefaultRouteTablePropagation = aws.String(d.Get("default_route_table_propagation").(string))
	}

	if d.HasChange("dns_support") {
		transitGatewayModified = true
		options.DnsSupport = aws.String(d.Get("dns_support").(string))
	}

	if d.HasChange("vpn_ecmp_support") {
		transitGatewayModified = true
		options.VpnEcmpSupport = aws.String(d.Get("vpn_ecmp_support").(string))
	}
	if transitGatewayModified {
		modifyTransitGatewayInput.TransitGatewayId = aws.String(d.Id())
		modifyTransitGatewayInput.Options = options
		if _, err := conn.ModifyTransitGateway(modifyTransitGatewayInput); err != nil {
			return fmt.Errorf("error updating EC2 Transit Gateway (%s) options: %s", d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating EC2 Transit Gateway (%s) tags: %s", d.Id(), err)
		}
	}

	return nil
}

func resourceAwsEc2TransitGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.DeleteTransitGatewayInput{
		TransitGatewayId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting EC2 Transit Gateway (%s): %s", d.Id(), input)
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteTransitGateway(input)

		if isAWSErr(err, "IncorrectState", "has non-deleted Transit Gateway Attachments") {
			return resource.RetryableError(err)
		}

		if isAWSErr(err, "IncorrectState", "has non-deleted DirectConnect Gateway Attachments") {
			return resource.RetryableError(err)
		}

		if isAWSErr(err, "IncorrectState", "has non-deleted VPN Attachments") {
			return resource.RetryableError(err)
		}

		if isAWSErr(err, "IncorrectState", "has non-deleted Transit Gateway Cross Region Peering Attachments") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isResourceTimeoutError(err) {
		_, err = conn.DeleteTransitGateway(input)
	}

	if isAWSErr(err, "InvalidTransitGatewayID.NotFound", "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting EC2 Transit Gateway: %s", err)
	}

	if err := waitForEc2TransitGatewayDeletion(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway (%s) deletion: %s", d.Id(), err)
	}

	return nil
}
