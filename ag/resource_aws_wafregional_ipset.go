package ag

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAwsWafRegionalIPSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalIPSetCreate,
		Read:   resourceAwsWafRegionalIPSetRead,
		Update: resourceAwsWafRegionalIPSetUpdate,
		Delete: resourceAwsWafRegionalIPSetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_set_descriptor": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRegionalIPSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateIPSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}
		return conn.CreateIPSet(params)
	})
	if err != nil {
		return err
	}
	resp := out.(*waf.CreateIPSetOutput)
	d.SetId(aws.StringValue(resp.IPSet.IPSetId))
	return resourceAwsWafRegionalIPSetUpdate(d, meta)
}

func resourceAwsWafRegionalIPSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	params := &waf.GetIPSetInput{
		IPSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetIPSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("ip_set_descriptor", flattenWafIpSetDescriptorWR(resp.IPSet.IPSetDescriptors))
	d.Set("name", resp.IPSet.Name)

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "waf-regional",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("ipset/%s", d.Id()),
	}
	d.Set("arn", arn.String())

	return nil
}

func flattenWafIpSetDescriptorWR(in []*waf.IPSetDescriptor) []interface{} {
	descriptors := make([]interface{}, len(in))

	for i, descriptor := range in {
		d := map[string]interface{}{
			"type":  *descriptor.Type,
			"value": *descriptor.Value,
		}
		descriptors[i] = d
	}

	return descriptors
}

func resourceAwsWafRegionalIPSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	if d.HasChange("ip_set_descriptor") {
		o, n := d.GetChange("ip_set_descriptor")
		oldD, newD := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateIPSetResourceWR(d.Id(), oldD, newD, conn, region)
		if err != nil {
			return fmt.Errorf("Error Updating WAF IPSet: %s", err)
		}
	}
	return resourceAwsWafRegionalIPSetRead(d, meta)
}

func resourceAwsWafRegionalIPSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	oldD := d.Get("ip_set_descriptor").(*schema.Set).List()

	if len(oldD) > 0 {
		noD := []interface{}{}
		err := updateIPSetResourceWR(d.Id(), oldD, noD, conn, region)

		if err != nil {
			return fmt.Errorf("Error Deleting IPSetDescriptors: %s", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteIPSetInput{
			ChangeToken: token,
			IPSetId:     aws.String(d.Id()),
		}
		log.Printf("[INFO] Deleting WAF IPSet")
		return conn.DeleteIPSet(req)
	})
	if err != nil {
		return fmt.Errorf("Error Deleting WAF IPSet: %s", err)
	}

	return nil
}

func updateIPSetResourceWR(id string, oldD, newD []interface{}, conn *wafregional.WAFRegional, region string) error {
	for _, ipSetUpdates := range diffWafIpSetDescriptors(oldD, newD) {
		wr := newWafRegionalRetryer(conn, region)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateIPSetInput{
				ChangeToken: token,
				IPSetId:     aws.String(id),
				Updates:     ipSetUpdates,
			}
			log.Printf("[INFO] Updating IPSet descriptor: %s", req)

			return conn.UpdateIPSet(req)
		})
		if err != nil {
			return fmt.Errorf("Error Updating WAF IPSet: %s", err)
		}
	}

	return nil
}
