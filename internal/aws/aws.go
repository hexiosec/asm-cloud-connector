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
	acm_t "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cloudfront_t "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_t "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elb_t "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambda_t "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	opensearch_t "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rds_t "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53_t "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3_t "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
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
	GetEC2Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetEIPResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetELBResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetS3Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetACMResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetRoute53Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetCloudFrontResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetAPIGatewayResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetAPIGatewayV2Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetEKSResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetRDSResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetOpenSearchResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
	GetLambdaResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error)
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

func (w *AWSWrapper) GetEC2Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
				tags := formatEC2Tags(instance.Tags, populateTags)
				if instance.PublicDnsName != nil {
					resources = appendAsset(resources, *instance.PublicDnsName, tags)
				}
				if instance.PublicIpAddress != nil {
					resources = appendAsset(resources, *instance.PublicIpAddress, tags)
				}
			}
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetEIPResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
	client := ec2.NewFromConfig(*w.cfg)
	logger.GetLogger(ctx).Trace().Msgf("getting Elastic IPs (EIP) resources")

	resp, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return resources, fmt.Errorf("aws: getting EIP resources, %w", err)
	}

	for _, address := range resp.Addresses {
		logger.GetLogger(ctx).Trace().Msgf("found address %s", *address.AllocationId)
		if address.PublicIp != nil {
			resources = appendAsset(resources, *address.PublicIp, formatEC2Tags(address.Tags, populateTags))
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetELBResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getELBTags(ctx, client, loadBalancer.LoadBalancerArn, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get ELB tags for %s", aws.ToString(loadBalancer.LoadBalancerArn))
			}
			if loadBalancer.DNSName != nil {
				resources = appendAsset(resources, *loadBalancer.DNSName, tags)
			}
		}

		if resp.NextMarker == nil {
			break
		}
		nextToken = resp.NextMarker
	}

	return resources, nil
}

func (w *AWSWrapper) GetS3Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getS3Tags(ctx, client, bucket.Name, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get S3 tags for bucket %s", aws.ToString(bucket.Name))
			}

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

			resources = appendAsset(resources, fmt.Sprintf(loc, *bucket.Name, w.cfg.Region), tags)
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

func (w *AWSWrapper) GetACMResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getACMTags(ctx, client, certificate.CertificateArn, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get ACM tags for %s", aws.ToString(certificate.CertificateArn))
			}
			if certificate.DomainName != nil {
				resources = appendAsset(resources, *certificate.DomainName, tags)
			}

			if certificate.HasAdditionalSubjectAlternativeNames == nil || !*certificate.HasAdditionalSubjectAlternativeNames {
				for _, name := range certificate.SubjectAlternativeNameSummaries {
					resources = appendAsset(resources, name, tags)
				}
			} else {
				detail, err := client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
					CertificateArn: certificate.CertificateArn,
				})
				if err != nil {
					logger.GetLogger(ctx).Warn().Err(err).Msgf("Failed to get %s certificate detail, unable to add subject alternative names", *certificate.CertificateArn)
				} else {
					for _, name := range detail.Certificate.SubjectAlternativeNames {
						resources = appendAsset(resources, name, tags)
					}
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

func (w *AWSWrapper) GetRoute53Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getRoute53HostedZoneTags(ctx, client, zone.Id, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get Route53 tags for hosted zone %s", aws.ToString(zone.Id))
			}

			resources = appendAsset(resources, *zone.Name, tags)

			resources, err = w.getHostedZoneResources(ctx, client, zone.Id, resources, tags)
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

func (w *AWSWrapper) getHostedZoneResources(ctx context.Context, client *route53.Client, zoneId *string, resources []*cloud_provider_t.Asset, tags []string) ([]*cloud_provider_t.Asset, error) {
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
			resources = appendAsset(resources, *record.Name, tags)
		}

		if resp.NextRecordIdentifier == nil {
			break
		}
		nextToken = resp.NextRecordIdentifier
	}

	return resources, nil
}

func (w *AWSWrapper) GetCloudFrontResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getCloudFrontTags(ctx, client, distribution.ARN, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get CloudFront tags for %s", aws.ToString(distribution.ARN))
			}
			resources = appendAsset(resources, *distribution.DomainName, tags)

			for _, origin := range distribution.Origins.Items {
				resources = appendAsset(resources, *origin.DomainName, tags)
			}
		}

		if resp.DistributionList.NextMarker == nil {
			break
		}
		nextToken = resp.DistributionList.NextMarker
	}

	return resources, nil
}

func (w *AWSWrapper) GetAPIGatewayResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			resources = appendAsset(resources, fmt.Sprintf("%s.execute-api.%s.amazonaws.com", *api.Id, w.cfg.Region), formatStringMapTags(api.Tags, populateTags))
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
			resources = appendAsset(resources, *domain.DomainName, formatStringMapTags(domain.Tags, populateTags))
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetAPIGatewayV2Resources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
				resources = appendAsset(resources, *api.ApiEndpoint, formatStringMapTags(api.Tags, populateTags))
			}
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	return resources, nil
}

func (w *AWSWrapper) GetEKSResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
				resources = appendAsset(resources, *detail.Cluster.Endpoint, formatStringMapTags(detail.Cluster.Tags, populateTags))
			}
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	return resources, nil
}

func (w *AWSWrapper) GetRDSResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getRDSTags(ctx, client, db.DBInstanceArn, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get RDS instance tags for %s", aws.ToString(db.DBInstanceArn))
			}

			if db.Endpoint != nil && db.Endpoint.Address != nil {
				resources = appendAsset(resources, *db.Endpoint.Address, tags)
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
			tags, tagErr := w.getRDSTags(ctx, client, db.DBClusterArn, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get RDS cluster tags for %s", aws.ToString(db.DBClusterArn))
			}

			if db.Endpoint != nil {
				resources = appendAsset(resources, *db.Endpoint, tags)
			}

			if db.ReaderEndpoint != nil {
				resources = appendAsset(resources, *db.ReaderEndpoint, tags)
			}
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetOpenSearchResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getOpenSearchTags(ctx, client, app.Arn, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get OpenSearch tags for %s", aws.ToString(app.Arn))
			}

			if app.Endpoint != nil {
				resources = appendAsset(resources, *app.Endpoint, tags)
			}
		}
	}

	return resources, nil
}

func (w *AWSWrapper) GetLambdaResources(ctx context.Context, resources []*cloud_provider_t.Asset, populateTags bool) ([]*cloud_provider_t.Asset, error) {
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
			tags, tagErr := w.getLambdaTags(ctx, client, function.FunctionArn, populateTags)
			if tagErr != nil {
				logger.GetLogger(ctx).Warn().Err(tagErr).Msgf("failed to get Lambda tags for %s", aws.ToString(function.FunctionArn))
			}

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

			resources = appendAsset(resources, *urlConfig.FunctionUrl, tags)
		}
	}

	return resources, nil
}

func appendAsset(resources []*cloud_provider_t.Asset, name string, tags []string) []*cloud_provider_t.Asset {
	if name == "" {
		return resources
	}

	var assetTags []string
	if len(tags) > 0 {
		assetTags = append([]string(nil), tags...)
	}

	return append(resources, &cloud_provider_t.Asset{Name: name, Tags: assetTags})
}

func formatEC2Tags(tags []ec2_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatACMTags(tags []acm_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatELBTags(tags []elb_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatS3Tags(tags []s3_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatRoute53Tags(tags []route53_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatCloudFrontTags(tags []cloudfront_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatOpenSearchTags(tags []opensearch_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatRDSTags(tags []rds_t.Tag, populateTags bool) []string {
	if !populateTags {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag.Key == nil || *tag.Key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, *tag.Key, aws.ToString(tag.Value))
	}

	return emptyToNil(formatted)
}

func formatStringMapTags(tags map[string]string, populateTags bool) []string {
	if !populateTags || len(tags) == 0 {
		return nil
	}

	formatted := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for key, value := range tags {
		if key == "" {
			continue
		}
		formatted = appendTag(formatted, seen, key, value)
	}

	return emptyToNil(formatted)
}

func appendTag(tags []string, seen map[string]struct{}, key string, value string) []string {
	formatted := fmt.Sprintf("%s: %s", key, value)
	if _, ok := seen[formatted]; ok {
		return tags
	}

	seen[formatted] = struct{}{}
	return append(tags, formatted)
}

func emptyToNil(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func (w *AWSWrapper) getELBTags(ctx context.Context, client *elb.Client, arn *string, populateTags bool) ([]string, error) {
	if !populateTags || arn == nil || *arn == "" {
		return nil, nil
	}

	resp, err := client.DescribeTags(ctx, &elb.DescribeTagsInput{
		ResourceArns: []string{*arn},
	})
	if err != nil {
		return nil, err
	}

	for _, description := range resp.TagDescriptions {
		if description.ResourceArn != nil && *description.ResourceArn == *arn {
			return formatELBTags(description.Tags, true), nil
		}
	}

	return nil, nil
}

func (w *AWSWrapper) getS3Tags(ctx context.Context, client *s3.Client, bucket *string, populateTags bool) ([]string, error) {
	if !populateTags || bucket == nil || *bucket == "" {
		return nil, nil
	}

	resp, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: bucket})
	if err != nil {
		if errType := (&smithy.GenericAPIError{}); errors.As(err, &errType) && errType.Code == "NoSuchTagSet" {
			return nil, nil
		}
		return nil, err
	}

	return formatS3Tags(resp.TagSet, true), nil
}

func (w *AWSWrapper) getACMTags(ctx context.Context, client *acm.Client, arn *string, populateTags bool) ([]string, error) {
	if !populateTags || arn == nil || *arn == "" {
		return nil, nil
	}

	resp, err := client.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{
		CertificateArn: arn,
	})
	if err != nil {
		return nil, err
	}

	return formatACMTags(resp.Tags, true), nil
}

func (w *AWSWrapper) getRoute53HostedZoneTags(ctx context.Context, client *route53.Client, zoneID *string, populateTags bool) ([]string, error) {
	if !populateTags || zoneID == nil || *zoneID == "" {
		return nil, nil
	}

	resourceID := strings.TrimPrefix(*zoneID, "/hostedzone/")
	resp, err := client.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
		ResourceId:   &resourceID,
		ResourceType: route53_t.TagResourceTypeHostedzone,
	})
	if err != nil {
		return nil, err
	}

	if resp.ResourceTagSet == nil {
		return nil, nil
	}

	return formatRoute53Tags(resp.ResourceTagSet.Tags, true), nil
}

func (w *AWSWrapper) getCloudFrontTags(ctx context.Context, client *cloudfront.Client, arn *string, populateTags bool) ([]string, error) {
	if !populateTags || arn == nil || *arn == "" {
		return nil, nil
	}

	resp, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
		Resource: arn,
	})
	if err != nil {
		return nil, err
	}

	if resp.Tags == nil {
		return nil, nil
	}

	return formatCloudFrontTags(resp.Tags.Items, true), nil
}

func (w *AWSWrapper) getRDSTags(ctx context.Context, client *rds.Client, arn *string, populateTags bool) ([]string, error) {
	if !populateTags || arn == nil || *arn == "" {
		return nil, nil
	}

	resp, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
		ResourceName: arn,
	})
	if err != nil {
		return nil, err
	}

	return formatRDSTags(resp.TagList, true), nil
}

func (w *AWSWrapper) getOpenSearchTags(ctx context.Context, client *opensearch.Client, arn *string, populateTags bool) ([]string, error) {
	if !populateTags || arn == nil || *arn == "" {
		return nil, nil
	}

	resp, err := client.ListTags(ctx, &opensearch.ListTagsInput{
		ARN: arn,
	})
	if err != nil {
		return nil, err
	}

	return formatOpenSearchTags(resp.TagList, true), nil
}

func (w *AWSWrapper) getLambdaTags(ctx context.Context, client *lambda.Client, arn *string, populateTags bool) ([]string, error) {
	if !populateTags || arn == nil || *arn == "" {
		return nil, nil
	}

	resp, err := client.ListTags(ctx, &lambda.ListTagsInput{
		Resource: arn,
	})
	if err != nil {
		return nil, err
	}

	return formatStringMapTags(resp.Tags, true), nil
}
