package ag

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAwsPinpointSMSChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPinpointSMSChannelUpsert,
		Read:   resourceAwsPinpointSMSChannelRead,
		Update: resourceAwsPinpointSMSChannelUpsert,
		Delete: resourceAwsPinpointSMSChannelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"sender_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"short_code": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"promotional_messages_per_second": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"transactional_messages_per_second": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceAwsPinpointSMSChannelUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	applicationId := d.Get("application_id").(string)

	params := &pinpoint.SMSChannelRequest{
		Enabled: aws.Bool(d.Get("enabled").(bool)),
	}

	if v, ok := d.GetOk("sender_id"); ok {
		params.SenderId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("short_code"); ok {
		params.ShortCode = aws.String(v.(string))
	}

	req := pinpoint.UpdateSmsChannelInput{
		ApplicationId:     aws.String(applicationId),
		SMSChannelRequest: params,
	}

	_, err := conn.UpdateSmsChannel(&req)
	if err != nil {
		return fmt.Errorf("error putting Pinpoint SMS Channel for application %s: %w", applicationId, err)
	}

	d.SetId(applicationId)

	return resourceAwsPinpointSMSChannelRead(d, meta)
}

func resourceAwsPinpointSMSChannelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[INFO] Reading Pinpoint SMS Channel  for application %s", d.Id())

	output, err := conn.GetSmsChannel(&pinpoint.GetSmsChannelInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint SMS Channel for application %s not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error getting Pinpoint SMS Channel for application %s: %w", d.Id(), err)
	}

	res := output.SMSChannelResponse
	d.Set("application_id", res.ApplicationId)
	d.Set("enabled", res.Enabled)
	d.Set("sender_id", res.SenderId)
	d.Set("short_code", res.ShortCode)
	d.Set("promotional_messages_per_second", res.PromotionalMessagesPerSecond)
	d.Set("transactional_messages_per_second", res.TransactionalMessagesPerSecond)
	return nil
}

func resourceAwsPinpointSMSChannelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[DEBUG] Deleting Pinpoint SMS Channel for application %s", d.Id())
	_, err := conn.DeleteSmsChannel(&pinpoint.DeleteSmsChannelInput{
		ApplicationId: aws.String(d.Id()),
	})

	if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Pinpoint SMS Channel for application %s: %w", d.Id(), err)
	}
	return nil
}
