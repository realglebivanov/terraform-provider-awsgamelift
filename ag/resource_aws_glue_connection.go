package ag

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAwsGlueConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGlueConnectionCreate,
		Read:   resourceAwsGlueConnectionRead,
		Update: resourceAwsGlueConnectionUpdate,
		Delete: resourceAwsGlueConnectionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"catalog_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},
			"connection_properties": {
				Type:         schema.TypeMap,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: MapKeyInSlice(glue.ConnectionPropertyKey_Values(), false),
				Elem:         &schema.Schema{Type: schema.TypeString},
			},
			"connection_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      glue.ConnectionTypeJdbc,
				ValidateFunc: validation.StringInSlice(glue.ConnectionType_Values(), false),
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 2048),
			},
			"match_criteria": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 10,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringLenBetween(1, 255),
				},
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
			},
			"physical_connection_requirements": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"availability_zone": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"security_group_id_list": {
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 50,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsGlueConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn
	var catalogID string
	if v, ok := d.GetOkExists("catalog_id"); ok {
		catalogID = v.(string)
	} else {
		catalogID = meta.(*AWSClient).accountid
	}
	name := d.Get("name").(string)

	input := &glue.CreateConnectionInput{
		CatalogId:       aws.String(catalogID),
		ConnectionInput: expandGlueConnectionInput(d),
	}

	log.Printf("[DEBUG] Creating Glue Connection: %s", input)
	_, err := conn.CreateConnection(input)
	if err != nil {
		return fmt.Errorf("error creating Glue Connection (%s): %w", name, err)
	}

	d.SetId(fmt.Sprintf("%s:%s", catalogID, name))

	return resourceAwsGlueConnectionRead(d, meta)
}

func resourceAwsGlueConnectionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	catalogID, connectionName, err := decodeGlueConnectionID(d.Id())
	if err != nil {
		return err
	}

	input := &glue.GetConnectionInput{
		CatalogId: aws.String(catalogID),
		Name:      aws.String(connectionName),
	}

	log.Printf("[DEBUG] Reading Glue Connection: %s", input)
	output, err := conn.GetConnection(input)
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			log.Printf("[WARN] Glue Connection (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Glue Connection (%s): %w", d.Id(), err)
	}

	connection := output.Connection
	if connection == nil {
		log.Printf("[WARN] Glue Connection (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	connectionArn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "glue",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("connection/%s", connectionName),
	}.String()
	d.Set("arn", connectionArn)

	d.Set("catalog_id", catalogID)
	if err := d.Set("connection_properties", aws.StringValueMap(connection.ConnectionProperties)); err != nil {
		return fmt.Errorf("error setting connection_properties: %w", err)
	}
	d.Set("connection_type", connection.ConnectionType)
	d.Set("description", connection.Description)
	if err := d.Set("match_criteria", flattenStringList(connection.MatchCriteria)); err != nil {
		return fmt.Errorf("error setting match_criteria: %w", err)
	}
	d.Set("name", connection.Name)
	if err := d.Set("physical_connection_requirements", flattenGluePhysicalConnectionRequirements(connection.PhysicalConnectionRequirements)); err != nil {
		return fmt.Errorf("error setting physical_connection_requirements: %w", err)
	}

	return nil
}

func resourceAwsGlueConnectionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	catalogID, connectionName, err := decodeGlueConnectionID(d.Id())
	if err != nil {
		return err
	}

	input := &glue.UpdateConnectionInput{
		CatalogId:       aws.String(catalogID),
		ConnectionInput: expandGlueConnectionInput(d),
		Name:            aws.String(connectionName),
	}

	log.Printf("[DEBUG] Updating Glue Connection: %s", input)
	_, err = conn.UpdateConnection(input)
	if err != nil {
		return fmt.Errorf("error updating Glue Connection (%s): %w", d.Id(), err)
	}

	return nil
}

func resourceAwsGlueConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	catalogID, connectionName, err := decodeGlueConnectionID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Deleting Glue Connection: %s", d.Id())
	err = deleteGlueConnection(conn, catalogID, connectionName)
	if err != nil {
		return fmt.Errorf("error deleting Glue Connection (%s): %w", d.Id(), err)
	}

	return nil
}

func decodeGlueConnectionID(id string) (string, string, error) {
	idParts := strings.Split(id, ":")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("expected ID in format CATALOG-ID:NAME, provided: %s", id)
	}
	return idParts[0], idParts[1], nil
}

func deleteGlueConnection(conn *glue.Glue, catalogID, connectionName string) error {
	input := &glue.DeleteConnectionInput{
		CatalogId:      aws.String(catalogID),
		ConnectionName: aws.String(connectionName),
	}

	_, err := conn.DeleteConnection(input)
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

func expandGlueConnectionInput(d *schema.ResourceData) *glue.ConnectionInput {
	connectionProperties := make(map[string]string)
	if val, ok := d.GetOkExists("connection_properties"); ok {
		for k, v := range val.(map[string]interface{}) {
			connectionProperties[k] = v.(string)
		}
	}

	connectionInput := &glue.ConnectionInput{
		ConnectionProperties: aws.StringMap(connectionProperties),
		ConnectionType:       aws.String(d.Get("connection_type").(string)),
		Name:                 aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		connectionInput.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("match_criteria"); ok {
		connectionInput.MatchCriteria = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("physical_connection_requirements"); ok {
		physicalConnectionRequirementsList := v.([]interface{})
		physicalConnectionRequirementsMap := physicalConnectionRequirementsList[0].(map[string]interface{})
		connectionInput.PhysicalConnectionRequirements = expandGluePhysicalConnectionRequirements(physicalConnectionRequirementsMap)
	}

	return connectionInput
}

func expandGluePhysicalConnectionRequirements(m map[string]interface{}) *glue.PhysicalConnectionRequirements {
	physicalConnectionRequirements := &glue.PhysicalConnectionRequirements{}

	if v, ok := m["availability_zone"]; ok {
		physicalConnectionRequirements.AvailabilityZone = aws.String(v.(string))
	}

	if v, ok := m["security_group_id_list"]; ok {
		physicalConnectionRequirements.SecurityGroupIdList = expandStringSet(v.(*schema.Set))
	}

	if v, ok := m["subnet_id"]; ok {
		physicalConnectionRequirements.SubnetId = aws.String(v.(string))
	}

	return physicalConnectionRequirements
}

func flattenGluePhysicalConnectionRequirements(physicalConnectionRequirements *glue.PhysicalConnectionRequirements) []map[string]interface{} {
	if physicalConnectionRequirements == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"availability_zone":      aws.StringValue(physicalConnectionRequirements.AvailabilityZone),
		"security_group_id_list": flattenStringSet(physicalConnectionRequirements.SecurityGroupIdList),
		"subnet_id":              aws.StringValue(physicalConnectionRequirements.SubnetId),
	}

	return []map[string]interface{}{m}
}
