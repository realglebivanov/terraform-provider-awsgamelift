package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/amplify"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/amplify/finder"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/tfresource"
)

func DomainAssociationStatus(conn *amplify.Amplify, appID, domainName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		domainAssociation, err := finder.DomainAssociationByAppIDAndDomainName(conn, appID, domainName)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return domainAssociation, aws.StringValue(domainAssociation.DomainStatus), nil
	}
}
