package ag

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
	tfec2 "github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/ec2"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/ec2/waiter"
)

func resourceAwsVpnGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpnGatewayCreate,
		Read:   resourceAwsVpnGatewayRead,
		Update: resourceAwsVpnGatewayUpdate,
		Delete: resourceAwsVpnGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"amazon_side_asn": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				ValidateFunc: validateAmazonSideAsn,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"tags": tagsSchema(),

			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsVpnGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	createOpts := &ec2.CreateVpnGatewayInput{
		AvailabilityZone:  aws.String(d.Get("availability_zone").(string)),
		Type:              aws.String(ec2.GatewayTypeIpsec1),
		TagSpecifications: ec2TagSpecificationsFromKeyValueTags(tags, ec2.ResourceTypeVpnGateway),
	}

	if asn, ok := d.GetOk("amazon_side_asn"); ok {
		i, err := strconv.ParseInt(asn.(string), 10, 64)
		if err != nil {
			return err
		}
		createOpts.AmazonSideAsn = aws.Int64(i)
	}

	// Create the VPN gateway
	log.Printf("[DEBUG] Creating VPN gateway")
	resp, err := conn.CreateVpnGateway(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating VPN gateway: %s", err)
	}

	d.SetId(aws.StringValue(resp.VpnGateway.VpnGatewayId))

	if _, ok := d.GetOk("vpc_id"); ok {
		if err := resourceAwsVpnGatewayAttach(d, meta); err != nil {
			return fmt.Errorf("error attaching EC2 VPN Gateway (%s) to VPC: %s", d.Id(), err)
		}
	}

	return resourceAwsVpnGatewayRead(d, meta)
}

func resourceAwsVpnGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	resp, err := conn.DescribeVpnGateways(&ec2.DescribeVpnGatewaysInput{
		VpnGatewayIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if isAWSErr(err, "InvalidVpnGatewayID.NotFound", "") {
			log.Printf("[WARN] VPC Gateway (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		} else {
			log.Printf("[ERROR] Error finding VpnGateway: %s", err)
			return err
		}
	}

	vpnGateway := resp.VpnGateways[0]
	if vpnGateway == nil || aws.StringValue(vpnGateway.State) == ec2.VpnStateDeleted {
		log.Printf("[WARN] VPC Gateway (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	vpnAttachment := vpnGatewayGetAttachment(vpnGateway)
	if vpnAttachment == nil {
		// Gateway exists but not attached to the VPC
		d.Set("vpc_id", "")
	} else {
		d.Set("vpc_id", vpnAttachment.VpcId)
	}

	if vpnGateway.AvailabilityZone != nil && aws.StringValue(vpnGateway.AvailabilityZone) != "" {
		d.Set("availability_zone", vpnGateway.AvailabilityZone)
	}
	d.Set("amazon_side_asn", strconv.FormatInt(aws.Int64Value(vpnGateway.AmazonSideAsn), 10))

	tags := keyvaluetags.Ec2KeyValueTags(vpnGateway.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   ec2.ServiceName,
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("vpn-gateway/%s", d.Id()),
	}.String()

	d.Set("arn", arn)

	return nil
}

func resourceAwsVpnGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("vpc_id") {
		// If we're already attached, detach it first
		if err := resourceAwsVpnGatewayDetach(d, meta); err != nil {
			return err
		}

		// Attach the VPN gateway to the new vpc
		if err := resourceAwsVpnGatewayAttach(d, meta); err != nil {
			return err
		}
	}

	conn := meta.(*AWSClient).ec2conn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating EC2 VPN Gateway (%s) tags: %s", d.Id(), err)
		}
	}

	return resourceAwsVpnGatewayRead(d, meta)
}

func resourceAwsVpnGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Detach if it is attached
	if err := resourceAwsVpnGatewayDetach(d, meta); err != nil {
		return err
	}

	log.Printf("[INFO] Deleting VPN gateway: %s", d.Id())
	input := &ec2.DeleteVpnGatewayInput{
		VpnGatewayId: aws.String(d.Id()),
	}
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteVpnGateway(input)
		if err == nil {
			return nil
		}

		if isAWSErr(err, "InvalidVpnGatewayID.NotFound", "") {
			return nil
		}
		if isAWSErr(err, "IncorrectState", "") {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteVpnGateway(input)
		if isAWSErr(err, "InvalidVpnGatewayID.NotFound", "") {
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("Error deleting VPN gateway: %s", err)
	}
	return nil
}

func resourceAwsVpnGatewayAttach(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	vpcId := d.Get("vpc_id").(string)

	if vpcId == "" {
		log.Printf("[DEBUG] Not attaching VPN Gateway '%s' as no VPC ID is set", d.Id())
		return nil
	}

	log.Printf(
		"[INFO] Attaching VPN Gateway '%s' to VPC '%s'",
		d.Id(),
		vpcId)

	req := &ec2.AttachVpnGatewayInput{
		VpnGatewayId: aws.String(d.Id()),
		VpcId:        aws.String(vpcId),
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.AttachVpnGateway(req)
		if err != nil {
			if isAWSErr(err, tfec2.InvalidVpnGatewayIDNotFound, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		_, err = conn.AttachVpnGateway(req)
	}

	if err != nil {
		return fmt.Errorf("Error attaching VPN gateway: %s", err)
	}

	// Wait for it to be fully attached before continuing
	log.Printf("[DEBUG] Waiting for VPN gateway (%s) to attach", d.Id())
	_, err = waiter.VpnGatewayVpcAttachmentAttached(conn, d.Id(), vpcId)

	if err != nil {
		return fmt.Errorf("error waiting for VPN Gateway (%s) Attachment (%s) to become attached: %w", d.Id(), vpcId, err)
	}

	return nil
}

func resourceAwsVpnGatewayDetach(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Get the old VPC ID to detach from
	vpcIdRaw, _ := d.GetChange("vpc_id")
	vpcId := vpcIdRaw.(string)

	if vpcId == "" {
		log.Printf(
			"[DEBUG] Not detaching VPN Gateway '%s' as no VPC ID is set",
			d.Id())
		return nil
	}

	log.Printf(
		"[INFO] Detaching VPN Gateway '%s' from VPC '%s'",
		d.Id(),
		vpcId)

	_, err := conn.DetachVpnGateway(&ec2.DetachVpnGatewayInput{
		VpnGatewayId: aws.String(d.Id()),
		VpcId:        aws.String(vpcId),
	})

	if isAWSErr(err, tfec2.InvalidVpnGatewayAttachmentNotFound, "") || isAWSErr(err, tfec2.InvalidVpnGatewayIDNotFound, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting VPN Gateway (%s) Attachment (%s): %w", d.Id(), vpcId, err)
	}

	// Wait for it to be fully detached before continuing
	_, err = waiter.VpnGatewayVpcAttachmentDetached(conn, d.Id(), vpcId)

	if err != nil {
		return fmt.Errorf("error waiting for VPN Gateway (%s) Attachment (%s) to become detached: %w", d.Id(), vpcId, err)
	}

	return nil
}

// vpnGatewayGetAttachment returns any VGW attachment that's in "attached" state or nil.
func vpnGatewayGetAttachment(vgw *ec2.VpnGateway) *ec2.VpcAttachment {
	for _, vpcAttachment := range vgw.VpcAttachments {
		if aws.StringValue(vpcAttachment.State) == ec2.AttachmentStatusAttached {
			return vpcAttachment
		}
	}
	return nil
}
