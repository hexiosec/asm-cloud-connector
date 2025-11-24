package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_t "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambda_t "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
)

type IAWSWrapper interface {
	AssumeRole(ctx context.Context, role string) (IAWSWrapper, error)
	ChangeRegion(region string)
	ResetRegion()
	CheckConnection(ctx context.Context) error
	GetSecretString(ctx context.Context, secret string) (string, error)
	ListAllAccounts(ctx context.Context) ([]string, error)
	GetRegions(ctx context.Context) ([]string, error)
	GetEC2Resources(ctx context.Context, resources []string) ([]string, error)
	GetEIPResources(ctx context.Context, resources []string) ([]string, error)
	GetELBResources(ctx context.Context, resources []string) ([]string, error)
	GetS3Resources(ctx context.Context, resources []string) ([]string, error)
	GetACMResources(ctx context.Context, resources []string) ([]string, error)
	GetRoute53Resources(ctx context.Context, resources []string) ([]string, error)
	GetCloudFrontResources(ctx context.Context, resources []string) ([]string, error)
	GetAPIGatewayResources(ctx context.Context, resources []string) ([]string, error)
	GetAPIGatewayV2Resources(ctx context.Context, resources []string) ([]string, error)
	GetEKSResources(ctx context.Context, resources []string) ([]string, error)
	GetRDSResources(ctx context.Context, resources []string) ([]string, error)
	GetOpenSearchResources(ctx context.Context, resources []string) ([]string, error)
	GetLambdaResources(ctx context.Context, resources []string) ([]string, error)
}

type AWSWrapper struct {
	cfg           *aws.Config
	defaultRegion string
}

func NewWrapper(ctx context.Context, region string) (IAWSWrapper, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("aws: unable to load SDK config, %w", err)
	}
	return &AWSWrapper{cfg: &cfg, defaultRegion: region}, nil
}

func (w *AWSWrapper) AssumeRole(ctx context.Context, role string) (IAWSWrapper, error) {
	client := sts.NewFromConfig(*w.cfg)

	provider := stscreds.NewAssumeRoleProvider(client, role)

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(w.defaultRegion),
		config.WithCredentialsProvider(provider),
	)
	if err != nil {
		return nil, fmt.Errorf("aws: unable to load SDK config with role %s, %w", role, err)
	}

	return &AWSWrapper{cfg: &cfg, defaultRegion: w.defaultRegion}, nil
}

func (w *AWSWrapper) ChangeRegion(region string) {
	w.cfg.Region = region
}

func (w *AWSWrapper) ResetRegion() {
	w.cfg.Region = w.defaultRegion
}

// Return nil if able to get the caller identity and the account is set
func (w *AWSWrapper) CheckConnection(ctx context.Context) error {
	client := sts.NewFromConfig(*w.cfg)

	resp, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("aws: get-caller-identity failed, %w", err)
	}

	if resp.Account == nil {
		return fmt.Errorf("aws: account not set")
	}

	return nil
}

func (w *AWSWrapper) GetSecretString(ctx context.Context, secret string) (string, error) {
	client := secretsmanager.NewFromConfig(*w.cfg)
	resp, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secret,
	})
	if err != nil {
		return "", fmt.Errorf("aws: getting secret, %w", err)
	}

	if resp.SecretString == nil {
		return "", fmt.Errorf("aws: secret string not set or secret is not formatted correctly")
	}

	return *resp.SecretString, nil
}

func (w *AWSWrapper) ListAllAccounts(ctx context.Context) ([]string, error) {
	client := organizations.NewFromConfig(*w.cfg)
	accounts := []string{}

	pager := organizations.NewListAccountsPaginator(client, &organizations.ListAccountsInput{})
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws: getting list accounts, %w", err)
		}
		for _, a := range page.Accounts {
			logger.GetLogger(ctx).Trace().Str("status", string(a.Status)).Msgf("found account %s", *a.Id)
			// Skip suspended accounts
			if a.Status == "SUSPENDED" {
				continue
			}
			accounts = append(accounts, *a.Id)
		}
	}

	return accounts, nil
}

func (w *AWSWrapper) GetRegions(ctx context.Context) ([]string, error) {
	client := ec2.NewFromConfig(*w.cfg)

	resp, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // only enabled regions
	})
	if err != nil {
		return nil, err
	}

	regions := make([]string, len(resp.Regions))
	for idx, region := range resp.Regions {
		regions[idx] = *region.RegionName
	}

	return regions, nil
}

func (w *AWSWrapper) GetEC2Resources(ctx context.Context, resources []string) ([]string, error) {
	client := ec2.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting EC2 VM resources")

	pager := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{
		Filters: []ec2_t.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	})

	for pager.HasMorePages() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting EC2 resources, %w", err)
		}

		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				logger.GetLogger(ctx).Trace().Msgf("found instance %s", *instance.InstanceId)
				if instance.PublicDnsName != nil {
					resources = append(resources, *instance.PublicDnsName)
				}
				if instance.PublicIpAddress != nil {
					resources = append(resources, *instance.PublicIpAddress)
				}
			}
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetEIPResources(ctx context.Context, resources []string) ([]string, error) {
	client := ec2.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting Elastic IPs (EIP) resources")

	pager := ec2.NewDescribeAddressesAttributePaginator(client, &ec2.DescribeAddressesAttributeInput{})

	for pager.HasMorePages() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting EIP resources, %w", err)
		}

		for _, address := range resp.Addresses {
			logger.GetLogger(ctx).Trace().Msgf("found address %s", *address.AllocationId)
			if address.PublicIp != nil {
				resources = append(resources, *address.PublicIp)
			}
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetELBResources(ctx context.Context, resources []string) ([]string, error) {
	client := elb.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting elastic load balancer (ELB) resources")

	var nextToken *string
	for {
		resp, err := client.DescribeLoadBalancers(
			ctx,
			&elb.DescribeLoadBalancersInput{
				Marker: nextToken,
			},
		)
		if err != nil {
			return resources, fmt.Errorf("aws: getting ELB resources, %w", err)
		}

		for _, loadBalancer := range resp.LoadBalancers {
			logger.GetLogger(ctx).Trace().Msgf("found load balancer %s", *loadBalancer.LoadBalancerArn)
			if loadBalancer.DNSName != nil {
				resources = append(resources, *loadBalancer.DNSName)
			}
		}

		if resp.NextMarker == nil {
			break
		}
		nextToken = resp.NextMarker
	}

	return resources, nil
}

func (w *AWSWrapper) GetS3Resources(ctx context.Context, resources []string) ([]string, error) {
	client := s3.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting S3 bucket resources")

	var nextToken *string
	for {
		resp, err := client.ListBuckets(
			ctx,
			&s3.ListBucketsInput{
				ContinuationToken: nextToken,
				BucketRegion:      &w.cfg.Region,
			},
		)
		if err != nil {
			return resources, fmt.Errorf("aws: getting S3 resources, %w", err)
		}

		for _, bucket := range resp.Buckets {
			logger.GetLogger(ctx).Trace().Msgf("found bucket %s", *bucket.Name)

			isPublic, err := w.isS3Public(ctx, client, bucket.Name)
			if err != nil {
				logger.GetLogger(ctx).Warn().Err(err).Msgf("failed to determine if %s bucket is public, assuming public", *bucket.Name)
				isPublic = true
			}

			if !isPublic {
				logger.GetLogger(ctx).Trace().Msgf("%s bucket is private, skipping", *bucket.Name)
				continue
			}

			isWebsite, err := w.isS3Website(ctx, client, bucket.Name)
			if err != nil {
				logger.GetLogger(ctx).Warn().Err(err).Msgf("failed to determine if %s bucket has website config, assuming no", *bucket.Name)
				isWebsite = false
			}

			loc := `%s.s3.%s.amazonaws.com`
			if isWebsite {
				loc = `%s.s3-website-%s.amazonaws.com`
			}

			resources = append(resources, fmt.Sprintf(loc, *bucket.Name, w.cfg.Region))
		}

		if resp.ContinuationToken == nil {
			break
		}
		nextToken = resp.ContinuationToken
	}

	return resources, nil
}

func (w *AWSWrapper) isS3Public(ctx context.Context, client *s3.Client, bucket *string) (bool, error) {
	// Check the Public access block
	{
		resp, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{Bucket: bucket})
		if err != nil {
			if errType := (&smithy.GenericAPIError{}); errors.As(err, &errType) && errType.Code == "NoSuchPublicAccessBlockConfiguration" {
				// Safe to continue — no config set
			} else {
				return false, err
			}
		} else if resp.PublicAccessBlockConfiguration != nil {
			blockPublicAcls := aws.ToBool(resp.PublicAccessBlockConfiguration.BlockPublicAcls)
			blockPublicPolicy := aws.ToBool(resp.PublicAccessBlockConfiguration.BlockPublicPolicy)
			ignorePublicAcls := aws.ToBool(resp.PublicAccessBlockConfiguration.IgnorePublicAcls)
			restrictPublicBuckets := aws.ToBool(resp.PublicAccessBlockConfiguration.RestrictPublicBuckets)

			// Public if any of the Public Access Block config is disabled
			if !blockPublicAcls || !blockPublicPolicy || !ignorePublicAcls || !restrictPublicBuckets {
				return true, nil
			}
		}
	}

	// Check the ACL
	{
		resp, err := client.GetBucketAcl(ctx, &s3.GetBucketAclInput{Bucket: bucket})
		if err != nil {
			return false, err
		}

		for _, grant := range resp.Grants {
			if grant.Grantee == nil {
				continue
			}

			if grant.Grantee.URI == nil {
				continue
			}

			if strings.Contains(*grant.Grantee.URI, "AllUsers") || strings.Contains(*grant.Grantee.URI, "AuthenticatedUsers") {
				return true, nil
			}
		}
	}

	// Check the policy
	{
		hasPolicy := true
		resp, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: bucket})
		if err != nil {
			if errType := (&smithy.GenericAPIError{}); errors.As(err, &errType) && errType.Code == "NoSuchBucketPolicy" {
				hasPolicy = false
			} else {
				return false, err
			}
		}

		if hasPolicy {
			// Parse the JSON Policy
			// {
			// 	"Statement": [{
			// 	"Principal": "*",
			// 	"Effect": "Deny",
			// 	}]
			// }
			policy := struct {
				Statement []*struct {
					Principal *json.RawMessage `json:"Principal,omitempty"` // Can be a string or object
					Effect    *string          `json:"Effect,omitempty"`
				} `json:"Statement"`
			}{}

			if err := json.Unmarshal([]byte(*resp.Policy), &policy); err != nil {
				return false, fmt.Errorf("aws: failed to parse bucket policy JSON %s, %w", *resp.Policy, err)
			}

			for _, stmt := range policy.Statement {
				if stmt == nil || stmt.Principal == nil || stmt.Effect == nil {
					continue
				}

				if string(*stmt.Principal) == "*" && *stmt.Effect == "Allow" {
					return true, nil
				}
			}
		}
	}

	// All checks failed, bucket is private
	return false, nil
}

func (w *AWSWrapper) isS3Website(ctx context.Context, client *s3.Client, bucket *string) (bool, error) {
	_, err := client.GetBucketWebsite(ctx, &s3.GetBucketWebsiteInput{Bucket: bucket})
	if err != nil {
		if errType := (&smithy.GenericAPIError{}); errors.As(err, &errType) && errType.Code == "NoSuchWebsiteConfiguration" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (w *AWSWrapper) GetACMResources(ctx context.Context, resources []string) ([]string, error) {
	client := acm.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting ACM TLS Certificate resources")

	var nextToken *string
	for {
		resp, err := client.ListCertificates(
			ctx,
			&acm.ListCertificatesInput{
				NextToken: nextToken,
			},
		)
		if err != nil {
			return resources, fmt.Errorf("aws: getting ACM resources, %w", err)
		}

		for _, certificate := range resp.CertificateSummaryList {
			logger.GetLogger(ctx).Trace().Msgf("found certificate %s", *certificate.CertificateArn)
			if certificate.DomainName != nil {
				resources = append(resources, *certificate.DomainName)
			}

			if certificate.HasAdditionalSubjectAlternativeNames == nil || !*certificate.HasAdditionalSubjectAlternativeNames {
				resources = append(resources, certificate.SubjectAlternativeNameSummaries...)
			} else {
				detail, err := client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
					CertificateArn: certificate.CertificateArn,
				})
				if err != nil {
					logger.GetLogger(ctx).Warn().Err(err).Msgf("Failed to get %s certificate detail, unable to add subject alternative names", *certificate.CertificateArn)
				} else {
					resources = append(resources, detail.Certificate.SubjectAlternativeNames...)
				}
			}

		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	return resources, nil
}

func (w *AWSWrapper) GetRoute53Resources(ctx context.Context, resources []string) ([]string, error) {
	client := route53.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting Route53 DNS resources")

	var nextToken *string
	for {
		resp, err := client.ListHostedZones(
			ctx,
			&route53.ListHostedZonesInput{
				Marker: nextToken,
			},
		)
		if err != nil {
			return resources, fmt.Errorf("aws: getting Route53 resources, %w", err)
		}

		for _, zone := range resp.HostedZones {
			logger.GetLogger(ctx).Trace().Msgf("found hosted zone %s", *zone.Id)

			resources = append(resources, *zone.Name)

			resources, err = w.getHostedZoneResources(ctx, client, zone.Id, resources)
			if err != nil {
				return resources, fmt.Errorf("aws: getting hosted zone %s, %w", *zone.Id, err)
			}
		}

		if resp.NextMarker == nil {
			break
		}
		nextToken = resp.NextMarker
	}

	return resources, nil
}

func (w *AWSWrapper) getHostedZoneResources(ctx context.Context, client *route53.Client, zoneId *string, resources []string) ([]string, error) {
	var nextToken *string
	for {
		resp, err := client.ListResourceRecordSets(
			ctx,
			&route53.ListResourceRecordSetsInput{
				HostedZoneId:          zoneId,
				StartRecordIdentifier: nextToken,
			},
		)
		if err != nil {
			return resources, err
		}

		// Only collect record names
		// We don’t need the record values (A, AAAA, etc.) here because ASM will resolve them itself.
		for _, record := range resp.ResourceRecordSets {
			resources = append(resources, *record.Name)
		}

		if resp.NextRecordIdentifier == nil {
			break
		}
		nextToken = resp.NextRecordIdentifier
	}

	return resources, nil
}

func (w *AWSWrapper) GetCloudFrontResources(ctx context.Context, resources []string) ([]string, error) {
	client := cloudfront.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting CloudFront CDN resources")

	var nextToken *string
	for {
		resp, err := client.ListDistributions(
			ctx,
			&cloudfront.ListDistributionsInput{
				Marker: nextToken,
			},
		)
		if err != nil {
			return resources, fmt.Errorf("aws: getting CloudFront resources, %w", err)
		}

		for _, distribution := range resp.DistributionList.Items {
			logger.GetLogger(ctx).Trace().Msgf("found distribution %s", *distribution.Id)
			resources = append(resources, *distribution.DomainName)

			for _, origin := range distribution.Origins.Items {
				resources = append(resources, *origin.DomainName)
			}
		}

		if resp.DistributionList.NextMarker == nil {
			break
		}
		nextToken = resp.DistributionList.NextMarker
	}

	return resources, nil
}

func (w *AWSWrapper) GetAPIGatewayResources(ctx context.Context, resources []string) ([]string, error) {
	client := apigateway.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting API Gateway resources")

	// Default AWS Managed domain name
	apiPager := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})
	for apiPager.HasMorePages() {

		resp, err := apiPager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting API Gateway resources, %w", err)
		}

		for _, api := range resp.Items {
			logger.GetLogger(ctx).Trace().Msgf("found api %s", *api.Id)
			resources = append(resources, fmt.Sprintf("%s.execute-api.%s.amazonaws.com", *api.Id, w.cfg.Region))
		}
	}

	// Custom Domain Names
	domainPager := apigateway.NewGetDomainNamesPaginator(client, &apigateway.GetDomainNamesInput{})
	for domainPager.HasMorePages() {

		resp, err := domainPager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting API Gateway Domain Name resources, %w", err)
		}

		for _, domain := range resp.Items {
			logger.GetLogger(ctx).Trace().Msgf("found domain %s", *domain.DomainNameId)
			resources = append(resources, *domain.DomainName)
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetAPIGatewayV2Resources(ctx context.Context, resources []string) ([]string, error) {
	client := apigatewayv2.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting API Gateway v2 resources")

	var nextToken *string
	for {
		resp, err := client.GetApis(
			ctx,
			&apigatewayv2.GetApisInput{
				NextToken: nextToken,
			},
		)
		if err != nil {
			return resources, fmt.Errorf("aws: getting API Gateway v2 resources, %w", err)
		}

		for _, api := range resp.Items {
			logger.GetLogger(ctx).Trace().Msgf("found api %s", *api.ApiId)
			if api.ApiEndpoint != nil {
				resources = append(resources, *api.ApiEndpoint)
			}
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	return resources, nil
}

func (w *AWSWrapper) GetEKSResources(ctx context.Context, resources []string) ([]string, error) {
	client := eks.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting EKS resources")

	var nextToken *string
	for {
		resp, err := client.ListClusters(
			ctx,
			&eks.ListClustersInput{
				NextToken: nextToken,
			},
		)
		if err != nil {
			return resources, fmt.Errorf("aws: getting EKS resources, %w", err)
		}

		for _, cluster := range resp.Clusters {
			logger.GetLogger(ctx).Trace().Msgf("found cluster %s", cluster)

			detail, err := client.DescribeCluster(
				ctx,
				&eks.DescribeClusterInput{
					Name: &cluster,
				},
			)
			if err != nil {
				return resources, fmt.Errorf("aws: getting EKS Cluster %s, %w", cluster, err)
			}

			if detail.Cluster != nil && detail.Cluster.Endpoint != nil {
				resources = append(resources, *detail.Cluster.Endpoint)
			}
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	return resources, nil
}

func (w *AWSWrapper) GetRDSResources(ctx context.Context, resources []string) ([]string, error) {
	client := rds.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting RDS Database resources")

	instancePager := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})
	for instancePager.HasMorePages() {
		resp, err := instancePager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting RDS Instance resources, %w", err)
		}

		for _, db := range resp.DBInstances {
			logger.GetLogger(ctx).Trace().Msgf("found db %s", *db.DBInstanceIdentifier)

			if db.Endpoint != nil && db.Endpoint.Address != nil {
				resources = append(resources, *db.Endpoint.Address)
			}
		}
	}

	clusterPager := rds.NewDescribeDBClustersPaginator(client, &rds.DescribeDBClustersInput{})
	for clusterPager.HasMorePages() {
		resp, err := clusterPager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting RDS Cluster resources, %w", err)
		}

		for _, db := range resp.DBClusters {
			logger.GetLogger(ctx).Trace().Msgf("found db %s", *db.DBClusterIdentifier)

			if db.Endpoint != nil {
				resources = append(resources, *db.Endpoint)
			}

			if db.ReaderEndpoint != nil {
				resources = append(resources, *db.ReaderEndpoint)
			}
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetOpenSearchResources(ctx context.Context, resources []string) ([]string, error) {
	client := opensearch.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting OpenSearch (ElasticSearch) resources")

	pager := opensearch.NewListApplicationsPaginator(client, &opensearch.ListApplicationsInput{})
	for pager.HasMorePages() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting OpenSearch resources, %w", err)
		}

		for _, app := range resp.ApplicationSummaries {
			logger.GetLogger(ctx).Trace().Msgf("found app %s", *app.Id)

			if app.Endpoint != nil {
				resources = append(resources, *app.Endpoint)
			}
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetLambdaResources(ctx context.Context, resources []string) ([]string, error) {
	client := lambda.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting Lambda Function resources")

	pager := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{})
	for pager.HasMorePages() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return resources, fmt.Errorf("aws: getting Lambda resources, %w", err)
		}

		for _, function := range resp.Functions {
			logger.GetLogger(ctx).Trace().Msgf("found function %s", *function.FunctionName)

			urlConfig, err := client.GetFunctionUrlConfig(
				ctx,
				&lambda.GetFunctionUrlConfigInput{
					FunctionName: function.FunctionName,
				},
			)
			if err != nil {
				if errType := (&lambda_t.ResourceNotFoundException{}); errors.As(err, &errType) {
					// Most functions don't have a URL config, can ignore
					continue
				}

				logger.GetLogger(ctx).Warn().Err(err).Msgf("Failed to get function url config, could not determine if function %s is publicly accessible", *function.FunctionName)
				continue
			}

			resources = append(resources, *urlConfig.FunctionUrl)
		}
	}

	return resources, nil
}
