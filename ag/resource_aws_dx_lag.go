package ag

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsDxLag() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxLagCreate,
		Read:   resourceAwsDxLagRead,
		Update: resourceAwsDxLagUpdate,
		Delete: resourceAwsDxLagDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"connections_bandwidth": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDxConnectionBandWidth(),
			},
			"location": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"jumbo_frame_capable": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
			"has_logical_redundancy": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsDxLagCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	req := &directconnect.CreateLagInput{
		ConnectionsBandwidth: aws.String(d.Get("connections_bandwidth").(string)),
		LagName:              aws.String(d.Get("name").(string)),
		Location:             aws.String(d.Get("location").(string)),
		NumberOfConnections:  aws.Int64(int64(1)),
	}

	if len(tags) > 0 {
		req.Tags = tags.IgnoreAws().DirectconnectTags()
	}

	log.Printf("[DEBUG] Creating Direct Connect LAG: %#v", req)
	resp, err := conn.CreateLag(req)
	if err != nil {
		return fmt.Errorf("error creating Direct Connect LAG: %s", err)
	}

	d.SetId(aws.StringValue(resp.LagId))

	// Delete unmanaged connection
	connectionID := aws.StringValue(resp.Connections[0].ConnectionId)
	deleteConnectionInput := &directconnect.DeleteConnectionInput{
		ConnectionId: resp.Connections[0].ConnectionId,
	}

	log.Printf("[DEBUG] Deleting newly created and unmanaged Direct Connect LAG (%s) Connection: %s", d.Id(), connectionID)
	if _, err := conn.DeleteConnection(deleteConnectionInput); err != nil {
		return fmt.Errorf("error deleting newly created and unmanaged Direct Connect LAG (%s) Connection (%s): %s", d.Id(), connectionID, err)
	}

	return resourceAwsDxLagRead(d, meta)
}

func resourceAwsDxLagRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	resp, err := conn.DescribeLags(&directconnect.DescribeLagsInput{
		LagId: aws.String(d.Id()),
	})
	if err != nil {
		if isNoSuchDxLagErr(err) {
			log.Printf("[WARN] Direct Connect LAG (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(resp.Lags) < 1 {
		log.Printf("[WARN] Direct Connect LAG (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if len(resp.Lags) != 1 {
		return fmt.Errorf("Number of Direct Connect LAGs (%s) isn't one, got %d", d.Id(), len(resp.Lags))
	}
	lag := resp.Lags[0]
	if d.Id() != aws.StringValue(lag.LagId) {
		return fmt.Errorf("Direct Connect LAG (%s) not found", d.Id())
	}

	if aws.StringValue(lag.LagState) == directconnect.LagStateDeleted {
		log.Printf("[WARN] Direct Connect LAG (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxlag/%s", d.Id()),
	}.String()
	d.Set("arn", arn)
	d.Set("name", lag.LagName)
	d.Set("connections_bandwidth", lag.ConnectionsBandwidth)
	d.Set("location", lag.Location)
	d.Set("jumbo_frame_capable", lag.JumboFrameCapable)
	d.Set("has_logical_redundancy", lag.HasLogicalRedundancy)

	tags, err := keyvaluetags.DirectconnectListTags(conn, arn)

	if err != nil {
		return fmt.Errorf("error listing tags for Direct Connect LAG (%s): %s", arn, err)
	}

	tags = tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceAwsDxLagUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	if d.HasChange("name") {
		req := &directconnect.UpdateLagInput{
			LagId:   aws.String(d.Id()),
			LagName: aws.String(d.Get("name").(string)),
		}

		log.Printf("[DEBUG] Updating Direct Connect LAG: %#v", req)
		_, err := conn.UpdateLag(req)
		if err != nil {
			return fmt.Errorf("error updating Direct Connect LAG (%s): %s", d.Id(), err)
		}
	}

	arn := d.Get("arn").(string)
	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.DirectconnectUpdateTags(conn, arn, o, n); err != nil {
			return fmt.Errorf("error updating Direct Connect LAG (%s) tags: %s", arn, err)
		}
	}

	return resourceAwsDxLagRead(d, meta)
}

func resourceAwsDxLagDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	if d.Get("force_destroy").(bool) {
		resp, err := conn.DescribeLags(&directconnect.DescribeLagsInput{
			LagId: aws.String(d.Id()),
		})
		if err != nil {
			if isNoSuchDxLagErr(err) {
				return nil
			}
			return err
		}

		if len(resp.Lags) < 1 {
			return nil
		}
		lag := resp.Lags[0]
		for _, v := range lag.Connections {
			log.Printf("[DEBUG] Deleting Direct Connect connection: %s", aws.StringValue(v.ConnectionId))
			_, err := conn.DeleteConnection(&directconnect.DeleteConnectionInput{
				ConnectionId: v.ConnectionId,
			})
			if err != nil && !isNoSuchDxConnectionErr(err) {
				return err
			}
		}
	}

	log.Printf("[DEBUG] Deleting Direct Connect LAG: %s", d.Id())
	_, err := conn.DeleteLag(&directconnect.DeleteLagInput{
		LagId: aws.String(d.Id()),
	})
	if err != nil {
		if isNoSuchDxLagErr(err) {
			return nil
		}
		return err
	}

	deleteStateConf := &resource.StateChangeConf{
		Pending:    []string{directconnect.LagStateAvailable, directconnect.LagStateRequested, directconnect.LagStatePending, directconnect.LagStateDeleting},
		Target:     []string{directconnect.LagStateDeleted},
		Refresh:    dxLagRefreshStateFunc(conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = deleteStateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect LAG (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func dxLagRefreshStateFunc(conn *directconnect.DirectConnect, lagId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &directconnect.DescribeLagsInput{
			LagId: aws.String(lagId),
		}
		resp, err := conn.DescribeLags(input)
		if err != nil {
			return nil, "failed", err
		}
		if len(resp.Lags) < 1 {
			return resp, directconnect.LagStateDeleted, nil
		}
		return resp, *resp.Lags[0].LagState, nil
	}
}

func isNoSuchDxLagErr(err error) bool {
	return isAWSErr(err, "DirectConnectClientException", "Could not find Lag with ID")
}
