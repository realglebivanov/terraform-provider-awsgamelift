package ag

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func Test_expandNetworkACLEntry(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  22,
			"to_port":    22,
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    1,
		},
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  443,
			"to_port":    443,
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    2,
		},
		map[string]interface{}{
			"protocol":   "-1",
			"from_port":  443,
			"to_port":    443,
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    2,
		},
	}
	expanded, _ := expandNetworkAclEntries(input, "egress")

	expected := []*ec2.NetworkAclEntry{
		{
			Protocol: aws.String("6"),
			PortRange: &ec2.PortRange{
				From: aws.Int64(22),
				To:   aws.Int64(22),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Int64(1),
			CidrBlock:  aws.String("0.0.0.0/0"),
			Egress:     aws.Bool(true),
		},
		{
			Protocol: aws.String("6"),
			PortRange: &ec2.PortRange{
				From: aws.Int64(443),
				To:   aws.Int64(443),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Int64(2),
			CidrBlock:  aws.String("0.0.0.0/0"),
			Egress:     aws.Bool(true),
		},
		{
			Protocol: aws.String("-1"),
			PortRange: &ec2.PortRange{
				From: aws.Int64(443),
				To:   aws.Int64(443),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Int64(2),
			CidrBlock:  aws.String("0.0.0.0/0"),
			Egress:     aws.Bool(true),
		},
	}

	if !reflect.DeepEqual(expanded, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			expanded,
			expected)
	}

}

func Test_validatePorts(t *testing.T) {
	for _, ts := range []struct {
		to       int64
		from     int64
		expected *expectedPortPair
		wanted   bool
	}{
		{0, 0, &expectedPortPair{0, 0}, true},
		{0, 1, &expectedPortPair{0, 0}, false},
	} {
		got := validatePorts(ts.to, ts.from, *ts.expected)
		if got != ts.wanted {
			t.Fatalf("Got: %t; Expected: %t\n", got, ts.wanted)
		}
	}
}
