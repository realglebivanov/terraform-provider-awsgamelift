package ag

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsRedshiftCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRedshiftClusterCreate,
		Read:   resourceAwsRedshiftClusterRead,
		Update: resourceAwsRedshiftClusterUpdate,
		Delete: resourceAwsRedshiftClusterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsRedshiftClusterImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(75 * time.Minute),
			Update: schema.DefaultTimeout(75 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"database_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 64),
					validation.StringMatch(regexp.MustCompile(`^[0-9a-z_$]+$`), "must contain only lowercase alphanumeric characters, underscores, and dollar signs"),
					validation.StringMatch(regexp.MustCompile(`(?i)^[a-z_]`), "first character must be a letter or underscore"),
				),
			},

			"cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringMatch(regexp.MustCompile(`^[0-9a-z-]+$`), "must contain only lowercase alphanumeric characters and hyphens"),
					validation.StringMatch(regexp.MustCompile(`(?i)^[a-z]`), "first character must be a letter"),
					validation.StringDoesNotMatch(regexp.MustCompile(`--`), "cannot contain two consecutive hyphens"),
					validation.StringDoesNotMatch(regexp.MustCompile(`-$`), "cannot end with a hyphen"),
				),
			},
			"cluster_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"node_type": {
				Type:     schema.TypeString,
				Required: true,
			},

			"master_username": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 128),
					validation.StringMatch(regexp.MustCompile(`^\w+$`), "must contain only alphanumeric characters"),
					validation.StringMatch(regexp.MustCompile(`(?i)^[a-z_]`), "first character must be a letter"),
				),
			},

			"master_password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(8, 64),
					validation.StringMatch(regexp.MustCompile(`^.*[a-z].*`), "must contain at least one lowercase letter"),
					validation.StringMatch(regexp.MustCompile(`^.*[A-Z].*`), "must contain at least one uppercase letter"),
					validation.StringMatch(regexp.MustCompile(`^.*[0-9].*`), "must contain at least one number"),
					validation.StringMatch(regexp.MustCompile(`^[^\@\/'" ]*$`), "cannot contain [/@\"' ]"),
				),
			},

			"cluster_security_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"cluster_subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					if val == nil {
						return ""
					}
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validateOnceAWeekWindowFormat,
			},

			"cluster_parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"automated_snapshot_retention_period": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				ValidateFunc: validation.IntAtMost(35),
			},

			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5439,
			},

			"cluster_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1.0",
			},

			"allow_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"number_of_nodes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			"publicly_accessible": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"enhanced_vpc_routing": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateArn,
			},

			"elastic_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"final_snapshot_identifier": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 255),
					validation.StringMatch(regexp.MustCompile(`^[0-9A-Za-z-]+$`), "must only contain alphanumeric characters and hyphens"),
					validation.StringDoesNotMatch(regexp.MustCompile(`--`), "cannot contain two consecutive hyphens"),
					validation.StringDoesNotMatch(regexp.MustCompile(`-$`), "cannot end in a hyphen"),
				),
			},

			"skip_final_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_public_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cluster_revision_number": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"iam_roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"snapshot_copy": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination_region": {
							Type:     schema.TypeString,
							Required: true,
						},
						"retention_period": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  7,
						},
						"grant_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"logging": {
				Type:             schema.TypeList,
				MaxItems:         1,
				Optional:         true,
				DiffSuppressFunc: suppressMissingOptionalConfigurationBlock,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enable": {
							Type:     schema.TypeBool,
							Required: true,
						},

						"bucket_name": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"s3_key_prefix": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},

			"snapshot_identifier": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"snapshot_cluster_identifier": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"owner_account": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsRedshiftClusterImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Neither skip_final_snapshot nor final_snapshot_identifier can be fetched
	// from any API call, so we need to default skip_final_snapshot to true so
	// that final_snapshot_identifier is not required
	d.Set("skip_final_snapshot", true)
	return []*schema.ResourceData{d}, nil
}

func resourceAwsRedshiftClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	if v, ok := d.GetOk("snapshot_identifier"); ok {
		restoreOpts := &redshift.RestoreFromClusterSnapshotInput{
			ClusterIdentifier:                aws.String(d.Get("cluster_identifier").(string)),
			SnapshotIdentifier:               aws.String(v.(string)),
			Port:                             aws.Int64(int64(d.Get("port").(int))),
			AllowVersionUpgrade:              aws.Bool(d.Get("allow_version_upgrade").(bool)),
			NodeType:                         aws.String(d.Get("node_type").(string)),
			PubliclyAccessible:               aws.Bool(d.Get("publicly_accessible").(bool)),
			AutomatedSnapshotRetentionPeriod: aws.Int64(int64(d.Get("automated_snapshot_retention_period").(int))),
		}

		if v, ok := d.GetOk("owner_account"); ok {
			restoreOpts.OwnerAccount = aws.String(v.(string))
		}

		if v, ok := d.GetOk("snapshot_cluster_identifier"); ok {
			restoreOpts.SnapshotClusterIdentifier = aws.String(v.(string))
		}

		if v, ok := d.GetOk("availability_zone"); ok {
			restoreOpts.AvailabilityZone = aws.String(v.(string))
		}

		if v, ok := d.GetOk("cluster_subnet_group_name"); ok {
			restoreOpts.ClusterSubnetGroupName = aws.String(v.(string))
		}

		if v, ok := d.GetOk("cluster_parameter_group_name"); ok {
			restoreOpts.ClusterParameterGroupName = aws.String(v.(string))
		}

		if v := d.Get("cluster_security_groups").(*schema.Set); v.Len() > 0 {
			restoreOpts.ClusterSecurityGroups = expandStringSet(v)
		}

		if v := d.Get("vpc_security_group_ids").(*schema.Set); v.Len() > 0 {
			restoreOpts.VpcSecurityGroupIds = expandStringSet(v)
		}

		if v, ok := d.GetOk("preferred_maintenance_window"); ok {
			restoreOpts.PreferredMaintenanceWindow = aws.String(v.(string))
		}

		if v, ok := d.GetOk("kms_key_id"); ok {
			restoreOpts.KmsKeyId = aws.String(v.(string))
		}

		if v, ok := d.GetOk("elastic_ip"); ok {
			restoreOpts.ElasticIp = aws.String(v.(string))
		}

		if v, ok := d.GetOk("enhanced_vpc_routing"); ok {
			restoreOpts.EnhancedVpcRouting = aws.Bool(v.(bool))
		}

		if v, ok := d.GetOk("iam_roles"); ok {
			restoreOpts.IamRoles = expandStringSet(v.(*schema.Set))
		}

		log.Printf("[DEBUG] Redshift Cluster restore cluster options: %s", restoreOpts)

		resp, err := conn.RestoreFromClusterSnapshot(restoreOpts)
		if err != nil {
			log.Printf("[ERROR] Error Restoring Redshift Cluster from Snapshot: %s", err)
			return err
		}

		d.SetId(aws.StringValue(resp.Cluster.ClusterIdentifier))

	} else {
		if _, ok := d.GetOk("master_password"); !ok {
			return fmt.Errorf(`provider.aws: aws_redshift_cluster: %s: "master_password": required field is not set`, d.Get("cluster_identifier").(string))
		}

		if _, ok := d.GetOk("master_username"); !ok {
			return fmt.Errorf(`provider.aws: aws_redshift_cluster: %s: "master_username": required field is not set`, d.Get("cluster_identifier").(string))
		}

		createOpts := &redshift.CreateClusterInput{
			ClusterIdentifier:                aws.String(d.Get("cluster_identifier").(string)),
			Port:                             aws.Int64(int64(d.Get("port").(int))),
			MasterUserPassword:               aws.String(d.Get("master_password").(string)),
			MasterUsername:                   aws.String(d.Get("master_username").(string)),
			ClusterVersion:                   aws.String(d.Get("cluster_version").(string)),
			NodeType:                         aws.String(d.Get("node_type").(string)),
			DBName:                           aws.String(d.Get("database_name").(string)),
			AllowVersionUpgrade:              aws.Bool(d.Get("allow_version_upgrade").(bool)),
			PubliclyAccessible:               aws.Bool(d.Get("publicly_accessible").(bool)),
			AutomatedSnapshotRetentionPeriod: aws.Int64(int64(d.Get("automated_snapshot_retention_period").(int))),
			Tags:                             tags.IgnoreAws().RedshiftTags(),
		}

		if v := d.Get("number_of_nodes").(int); v > 1 {
			createOpts.ClusterType = aws.String("multi-node")
			createOpts.NumberOfNodes = aws.Int64(int64(d.Get("number_of_nodes").(int)))
		} else {
			createOpts.ClusterType = aws.String("single-node")
		}

		if v := d.Get("cluster_security_groups").(*schema.Set); v.Len() > 0 {
			createOpts.ClusterSecurityGroups = expandStringSet(v)
		}

		if v := d.Get("vpc_security_group_ids").(*schema.Set); v.Len() > 0 {
			createOpts.VpcSecurityGroupIds = expandStringSet(v)
		}

		if v, ok := d.GetOk("cluster_subnet_group_name"); ok {
			createOpts.ClusterSubnetGroupName = aws.String(v.(string))
		}

		if v, ok := d.GetOk("availability_zone"); ok {
			createOpts.AvailabilityZone = aws.String(v.(string))
		}

		if v, ok := d.GetOk("preferred_maintenance_window"); ok {
			createOpts.PreferredMaintenanceWindow = aws.String(v.(string))
		}

		if v, ok := d.GetOk("cluster_parameter_group_name"); ok {
			createOpts.ClusterParameterGroupName = aws.String(v.(string))
		}

		if v, ok := d.GetOk("encrypted"); ok {
			createOpts.Encrypted = aws.Bool(v.(bool))
		}

		if v, ok := d.GetOk("enhanced_vpc_routing"); ok {
			createOpts.EnhancedVpcRouting = aws.Bool(v.(bool))
		}

		if v, ok := d.GetOk("kms_key_id"); ok {
			createOpts.KmsKeyId = aws.String(v.(string))
		}

		if v, ok := d.GetOk("elastic_ip"); ok {
			createOpts.ElasticIp = aws.String(v.(string))
		}

		if v, ok := d.GetOk("iam_roles"); ok {
			createOpts.IamRoles = expandStringSet(v.(*schema.Set))
		}

		log.Printf("[DEBUG] Redshift Cluster create options: %s", createOpts)
		resp, err := conn.CreateCluster(createOpts)
		if err != nil {
			log.Printf("[ERROR] Error creating Redshift Cluster: %s", err)
			return err
		}

		log.Printf("[DEBUG]: Cluster create response: %s", resp)
		d.SetId(aws.StringValue(resp.Cluster.ClusterIdentifier))
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying", "restoring", "available, prep-for-resize"},
		Target:     []string{"available"},
		Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Redshift Cluster state to be \"available\": %s", err)
	}

	if v, ok := d.GetOk("snapshot_copy"); ok {
		err := enableRedshiftSnapshotCopy(d.Id(), v.([]interface{}), conn)
		if err != nil {
			return err
		}
	}

	if _, ok := d.GetOk("logging.0.enable"); ok {
		if err := enableRedshiftClusterLogging(d, conn); err != nil {
			return fmt.Errorf("error enabling Redshift Cluster (%s) logging: %s", d.Id(), err)
		}
	}

	return resourceAwsRedshiftClusterRead(d, meta)
}

func resourceAwsRedshiftClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	log.Printf("[INFO] Reading Redshift Cluster Information: %s", d.Id())
	resp, err := conn.DescribeClusters(&redshift.DescribeClustersInput{
		ClusterIdentifier: aws.String(d.Id()),
	})

	if err != nil {
		if isAWSErr(err, redshift.ErrCodeClusterNotFoundFault, "") {
			d.SetId("")
			log.Printf("[DEBUG] Redshift Cluster (%s) not found", d.Id())
			return nil
		}
		log.Printf("[DEBUG] Error describing Redshift Cluster (%s)", d.Id())
		return err
	}

	var rsc *redshift.Cluster
	for _, c := range resp.Clusters {
		if *c.ClusterIdentifier == d.Id() {
			rsc = c
		}
	}

	if rsc == nil {
		log.Printf("[WARN] Redshift Cluster (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] Reading Redshift Cluster Logging Status: %s", d.Id())
	loggingStatus, loggingErr := conn.DescribeLoggingStatus(&redshift.DescribeLoggingStatusInput{
		ClusterIdentifier: aws.String(d.Id()),
	})

	if loggingErr != nil {
		return loggingErr
	}

	d.Set("master_username", rsc.MasterUsername)
	d.Set("node_type", rsc.NodeType)
	d.Set("allow_version_upgrade", rsc.AllowVersionUpgrade)
	d.Set("database_name", rsc.DBName)
	d.Set("cluster_identifier", rsc.ClusterIdentifier)
	d.Set("cluster_version", rsc.ClusterVersion)

	d.Set("cluster_subnet_group_name", rsc.ClusterSubnetGroupName)
	d.Set("availability_zone", rsc.AvailabilityZone)
	d.Set("encrypted", rsc.Encrypted)
	d.Set("enhanced_vpc_routing", rsc.EnhancedVpcRouting)
	d.Set("kms_key_id", rsc.KmsKeyId)
	d.Set("automated_snapshot_retention_period", rsc.AutomatedSnapshotRetentionPeriod)
	d.Set("preferred_maintenance_window", rsc.PreferredMaintenanceWindow)
	if rsc.Endpoint != nil && rsc.Endpoint.Address != nil {
		endpoint := *rsc.Endpoint.Address
		if rsc.Endpoint.Port != nil {
			endpoint = fmt.Sprintf("%s:%d", endpoint, *rsc.Endpoint.Port)
		}
		d.Set("dns_name", rsc.Endpoint.Address)
		d.Set("port", rsc.Endpoint.Port)
		d.Set("endpoint", endpoint)
	}
	d.Set("cluster_parameter_group_name", rsc.ClusterParameterGroups[0].ParameterGroupName)
	if len(rsc.ClusterNodes) > 1 {
		d.Set("cluster_type", "multi-node")
	} else {
		d.Set("cluster_type", "single-node")
	}
	d.Set("number_of_nodes", rsc.NumberOfNodes)
	d.Set("publicly_accessible", rsc.PubliclyAccessible)

	var vpcg []string
	for _, g := range rsc.VpcSecurityGroups {
		vpcg = append(vpcg, *g.VpcSecurityGroupId)
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("Error saving VPC Security Group IDs to state for Redshift Cluster (%s): %s", d.Id(), err)
	}

	var csg []string
	for _, g := range rsc.ClusterSecurityGroups {
		csg = append(csg, *g.ClusterSecurityGroupName)
	}
	if err := d.Set("cluster_security_groups", csg); err != nil {
		return fmt.Errorf("Error saving Cluster Security Group Names to state for Redshift Cluster (%s): %s", d.Id(), err)
	}

	var iamRoles []string
	for _, i := range rsc.IamRoles {
		iamRoles = append(iamRoles, *i.IamRoleArn)
	}
	if err := d.Set("iam_roles", iamRoles); err != nil {
		return fmt.Errorf("Error saving IAM Roles to state for Redshift Cluster (%s): %s", d.Id(), err)
	}

	d.Set("cluster_public_key", rsc.ClusterPublicKey)
	d.Set("cluster_revision_number", rsc.ClusterRevisionNumber)
	tags := keyvaluetags.RedshiftKeyValueTags(rsc.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	d.Set("snapshot_copy", flattenRedshiftSnapshotCopy(rsc.ClusterSnapshotCopyStatus))

	if err := d.Set("logging", flattenRedshiftLogging(loggingStatus)); err != nil {
		return fmt.Errorf("error setting logging: %s", err)
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "redshift",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("cluster:%s", d.Id()),
	}.String()

	d.Set("arn", arn)

	return nil
}

func resourceAwsRedshiftClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.RedshiftUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating Redshift Cluster (%s) tags: %s", d.Get("arn").(string), err)
		}
	}

	requestUpdate := false
	log.Printf("[INFO] Building Redshift Modify Cluster Options")
	req := &redshift.ModifyClusterInput{
		ClusterIdentifier: aws.String(d.Id()),
	}

	// If the cluster type, node type, or number of nodes changed, then the AWS API expects all three
	// items to be sent over
	if d.HasChanges("cluster_type", "node_type", "number_of_nodes") {
		req.ClusterType = aws.String(d.Get("cluster_type").(string))
		req.NodeType = aws.String(d.Get("node_type").(string))
		if v := d.Get("number_of_nodes").(int); v > 1 {
			req.ClusterType = aws.String("multi-node")
			req.NumberOfNodes = aws.Int64(int64(d.Get("number_of_nodes").(int)))
		} else {
			req.ClusterType = aws.String("single-node")
		}
		requestUpdate = true
	}

	if d.HasChange("cluster_security_groups") {
		req.ClusterSecurityGroups = expandStringSet(d.Get("cluster_security_groups").(*schema.Set))
		requestUpdate = true
	}

	if d.HasChange("vpc_security_group_ids") {
		req.VpcSecurityGroupIds = expandStringSet(d.Get("vpc_security_group_ids").(*schema.Set))
		requestUpdate = true
	}

	if d.HasChange("master_password") {
		req.MasterUserPassword = aws.String(d.Get("master_password").(string))
		requestUpdate = true
	}

	if d.HasChange("cluster_parameter_group_name") {
		req.ClusterParameterGroupName = aws.String(d.Get("cluster_parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("automated_snapshot_retention_period") {
		req.AutomatedSnapshotRetentionPeriod = aws.Int64(int64(d.Get("automated_snapshot_retention_period").(int)))
		requestUpdate = true
	}

	if d.HasChange("preferred_maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("cluster_version") {
		req.ClusterVersion = aws.String(d.Get("cluster_version").(string))
		requestUpdate = true
	}

	if d.HasChange("allow_version_upgrade") {
		req.AllowVersionUpgrade = aws.Bool(d.Get("allow_version_upgrade").(bool))
		requestUpdate = true
	}

	if d.HasChange("publicly_accessible") {
		req.PubliclyAccessible = aws.Bool(d.Get("publicly_accessible").(bool))
		requestUpdate = true
	}

	if d.HasChange("enhanced_vpc_routing") {
		req.EnhancedVpcRouting = aws.Bool(d.Get("enhanced_vpc_routing").(bool))
		requestUpdate = true
	}

	if d.HasChange("encrypted") {
		req.Encrypted = aws.Bool(d.Get("encrypted").(bool))
		requestUpdate = true
	}

	if d.Get("encrypted").(bool) && d.HasChange("kms_key_id") {
		req.KmsKeyId = aws.String(d.Get("kms_key_id").(string))
		requestUpdate = true
	}

	if requestUpdate {
		log.Printf("[INFO] Modifying Redshift Cluster: %s", d.Id())
		log.Printf("[DEBUG] Redshift Cluster Modify options: %s", req)
		_, err := conn.ModifyCluster(req)
		if err != nil {
			return fmt.Errorf("Error modifying Redshift Cluster (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("iam_roles") {
		o, n := d.GetChange("iam_roles")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		removeIams := os.Difference(ns)
		addIams := ns.Difference(os)

		log.Printf("[INFO] Building Redshift Modify Cluster IAM Role Options")
		req := &redshift.ModifyClusterIamRolesInput{
			ClusterIdentifier: aws.String(d.Id()),
			AddIamRoles:       expandStringSet(addIams),
			RemoveIamRoles:    expandStringSet(removeIams),
		}

		log.Printf("[INFO] Modifying Redshift Cluster IAM Roles: %s", d.Id())
		log.Printf("[DEBUG] Redshift Cluster Modify IAM Role options: %s", req)
		_, err := conn.ModifyClusterIamRoles(req)
		if err != nil {
			return fmt.Errorf("Error modifying Redshift Cluster IAM Roles (%s): %s", d.Id(), err)
		}
	}

	if requestUpdate || d.HasChange("iam_roles") {

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"creating", "deleting", "rebooting", "resizing", "renaming", "modifying", "available, prep-for-resize"},
			Target:     []string{"available"},
			Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(d.Id(), conn),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
		}

		// Wait, catching any errors
		_, err := stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error Modifying Redshift Cluster (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("snapshot_copy") {
		if v, ok := d.GetOk("snapshot_copy"); ok {
			err := enableRedshiftSnapshotCopy(d.Id(), v.([]interface{}), conn)
			if err != nil {
				return err
			}
		} else {
			_, err := conn.DisableSnapshotCopy(&redshift.DisableSnapshotCopyInput{
				ClusterIdentifier: aws.String(d.Id()),
			})
			if err != nil {
				return fmt.Errorf("Failed to disable snapshot copy: %s", err)
			}
		}
	}

	if d.HasChange("logging") {
		if loggingEnabled, ok := d.GetOk("logging.0.enable"); ok && loggingEnabled.(bool) {
			log.Printf("[INFO] Enabling Logging for Redshift Cluster %q", d.Id())
			err := enableRedshiftClusterLogging(d, conn)
			if err != nil {
				return err
			}
		} else {
			log.Printf("[INFO] Disabling Logging for Redshift Cluster %q", d.Id())
			_, err := conn.DisableLogging(&redshift.DisableLoggingInput{
				ClusterIdentifier: aws.String(d.Id()),
			})
			if err != nil {
				return err
			}
		}
	}

	return resourceAwsRedshiftClusterRead(d, meta)
}

func enableRedshiftClusterLogging(d *schema.ResourceData, conn *redshift.Redshift) error {
	bucketNameRaw, ok := d.GetOk("logging.0.bucket_name")

	if !ok {
		return fmt.Errorf("bucket_name must be set when enabling logging for Redshift Clusters")
	}

	params := &redshift.EnableLoggingInput{
		ClusterIdentifier: aws.String(d.Id()),
		BucketName:        aws.String(bucketNameRaw.(string)),
	}

	if v, ok := d.GetOk("logging.0.s3_key_prefix"); ok {
		params.S3KeyPrefix = aws.String(v.(string))
	}

	if _, err := conn.EnableLogging(params); err != nil {
		return fmt.Errorf("error enabling Redshift Cluster (%s) logging: %s", d.Id(), err)
	}
	return nil
}

func enableRedshiftSnapshotCopy(id string, scList []interface{}, conn *redshift.Redshift) error {
	sc := scList[0].(map[string]interface{})

	input := redshift.EnableSnapshotCopyInput{
		ClusterIdentifier: aws.String(id),
		DestinationRegion: aws.String(sc["destination_region"].(string)),
	}
	if rp, ok := sc["retention_period"]; ok {
		input.RetentionPeriod = aws.Int64(int64(rp.(int)))
	}
	if gn, ok := sc["grant_name"]; ok {
		input.SnapshotCopyGrantName = aws.String(gn.(string))
	}

	_, err := conn.EnableSnapshotCopy(&input)
	if err != nil {
		return fmt.Errorf("Failed to enable snapshot copy: %s", err)
	}
	return nil
}

func resourceAwsRedshiftClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	log.Printf("[DEBUG] Destroying Redshift Cluster (%s)", d.Id())

	deleteOpts := redshift.DeleteClusterInput{
		ClusterIdentifier: aws.String(d.Id()),
	}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	deleteOpts.SkipFinalClusterSnapshot = aws.Bool(skipFinalSnapshot)

	if !skipFinalSnapshot {
		if name, present := d.GetOk("final_snapshot_identifier"); present {
			deleteOpts.FinalClusterSnapshotIdentifier = aws.String(name.(string))
		} else {
			return fmt.Errorf("Redshift Cluster Instance FinalSnapshotIdentifier is required when a final snapshot is required")
		}
	}

	log.Printf("[DEBUG] Deleting Redshift Cluster: %s", deleteOpts)
	log.Printf("[DEBUG] schema.TimeoutDelete: %+v", d.Timeout(schema.TimeoutDelete))
	err := deleteAwsRedshiftCluster(&deleteOpts, conn, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return err
	}

	log.Printf("[INFO] Redshift Cluster %s successfully deleted", d.Id())

	return nil
}

func deleteAwsRedshiftCluster(opts *redshift.DeleteClusterInput, conn *redshift.Redshift, timeout time.Duration) error {
	id := *opts.ClusterIdentifier
	log.Printf("[INFO] Deleting Redshift Cluster %q", id)
	err := resource.Retry(15*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteCluster(opts)
		if isAWSErr(err, redshift.ErrCodeInvalidClusterStateFault, "") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteCluster(opts)
	}
	if err != nil {
		return fmt.Errorf("Error deleting Redshift Cluster (%s): %s", id, err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"available", "creating", "deleting", "rebooting", "resizing", "renaming", "final-snapshot"},
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(id, conn),
		Timeout:    timeout,
		MinTimeout: 5 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func resourceAwsRedshiftClusterStateRefreshFunc(id string, conn *redshift.Redshift) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[INFO] Reading Redshift Cluster Information: %s", id)
		resp, err := conn.DescribeClusters(&redshift.DescribeClustersInput{
			ClusterIdentifier: aws.String(id),
		})

		if err != nil {
			if isAWSErr(err, redshift.ErrCodeClusterNotFoundFault, "") {
				return 42, "destroyed", nil
			}
			log.Printf("[WARN] Error on retrieving Redshift Cluster (%s) when waiting: %s", id, err)
			return nil, "", err
		}

		var rsc *redshift.Cluster

		for _, c := range resp.Clusters {
			if *c.ClusterIdentifier == id {
				rsc = c
			}
		}

		if rsc == nil {
			return 42, "destroyed", nil
		}

		if rsc.ClusterStatus != nil {
			log.Printf("[DEBUG] Redshift Cluster status (%s): %s", id, *rsc.ClusterStatus)
		}

		return rsc, *rsc.ClusterStatus, nil
	}
}
