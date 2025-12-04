package gcp

import (
	"context"
	"errors"
	"fmt"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/asset/apiv1/assetpb"
	certificatemanager "cloud.google.com/go/certificatemanager/apiv1"
	certificatemanagerpb "cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"cloud.google.com/go/storage"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"google.golang.org/api/iterator"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

type IGCPWrapper interface {
	CheckConnection(ctx context.Context) error
	GetAssets(ctx context.Context, project string, assetTypes []string) ([]*assetpb.Asset, error)
	GetCertificates(ctx context.Context, project string) ([]*certificatemanagerpb.Certificate, error)
	IsBucketPublic(ctx context.Context, bucketName string) bool
}
type GCPWrapper struct {
}

func NewWrapper() (IGCPWrapper, error) {
	return &GCPWrapper{}, nil
}

// Return nil if able to create any client and therefore can authenticate
// doesn't check that the required permissions are set
func (w *GCPWrapper) CheckConnection(ctx context.Context) error {
	c, err := asset.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("gcp: failed to create client, %w", err)
	}
	c.Close()
	return nil
}

func (w *GCPWrapper) GetAssets(ctx context.Context, project string, assetTypes []string) ([]*assetpb.Asset, error) {
	c, err := asset.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: failed to create client, %w", err)
	}
	defer c.Close()

	it := c.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Parent:      project,
		ContentType: assetpb.ContentType_RESOURCE,
		AssetTypes:  assetTypes,
	})
	var assets []*assetpb.Asset

	for {
		a, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isServiceDisabledErr(err) {
				// Warn and continue
				logger.GetLogger(ctx).Warn().
					Msg("assets API disabled — skipping asset discovery")
				return assets, nil
			}
			return nil, err
		}

		assets = append(assets, a)
	}

	return assets, nil
}

// GetCertificates retrieves Certificate Manager resources for processing.
// These are not available in Cloud Asset Inventory, so we must query
// certificatemanager.googleapis.com directly.
func (w *GCPWrapper) GetCertificates(ctx context.Context, project string) ([]*certificatemanagerpb.Certificate, error) {
	client, err := certificatemanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: failed to create certificate manager client: %w", err)
	}
	defer client.Close()

	// "-" means all locations
	it := client.ListCertificates(ctx, &certificatemanagerpb.ListCertificatesRequest{
		Parent: fmt.Sprintf("%s/locations/-", project),
	})
	var certificates []*certificatemanagerpb.Certificate

	for {
		cert, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isServiceDisabledErr(err) {
				// Warn and continue
				logger.GetLogger(ctx).Warn().
					Msg("Certificate Manager API disabled — skipping certificate discovery")
				return []*certificatemanagerpb.Certificate{}, nil
			}
			return nil, fmt.Errorf("gcp: failed to list certificates: %w", err)
		}

		certificates = append(certificates, cert)
	}

	return certificates, nil
}

func (w *GCPWrapper) isBucketPolicyPublic(ctx context.Context, bucket *storage.BucketHandle) (bool, error) {
	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		return false, err
	}

	if policy == nil {
		return false, fmt.Errorf("gcp: policy nil")
	}

	for _, role := range policy.Roles() {
		for _, member := range policy.Members(role) {
			if member == "allUsers" || member == "allAuthenticatedUsers" {
				return true, nil
			}
		}
	}

	return false, nil
}

func (w *GCPWrapper) isBucketACLPublic(ctx context.Context, bucket *storage.BucketHandle) (bool, error) {
	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		return false, err
	}

	for _, ac := range attrs.ACL {
		if ac.Entity == storage.AllUsers || ac.Entity == storage.AllAuthenticatedUsers {
			return true, nil
		}
	}

	for _, ac := range attrs.DefaultObjectACL {
		if ac.Entity == storage.AllUsers || ac.Entity == storage.AllAuthenticatedUsers {
			return true, nil
		}
	}

	return false, nil
}

func (w *GCPWrapper) IsBucketPublic(ctx context.Context, bucketName string) bool {
	iCtx := logger.WithLogger(ctx, logger.GetLogger(ctx).With().Str("bucket", bucketName).Logger())

	sc, err := storage.NewClient(iCtx)
	if err != nil {
		logger.GetLogger(iCtx).Warn().Err(err).Msg("failed to create storage client — assuming bucket is not public")
		return false
	}
	defer sc.Close()

	bucket := sc.Bucket(bucketName)

	policyPublic, err := w.isBucketPolicyPublic(iCtx, bucket)
	if err != nil {
		logger.GetLogger(iCtx).Warn().Err(err).Msg("failed to check IAM policy - trying bucket ACL")
	} else if policyPublic {
		return true
	}

	aclPublic, err := w.isBucketACLPublic(iCtx, bucket)
	if err != nil {
		logger.GetLogger(iCtx).Warn().Err(err).Msg("failed to check ACL - assuming bucket is not public")
		return false
	}

	return aclPublic
}

func isServiceDisabledErr(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	for _, detail := range st.Details() {
		ei, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}

		// Check if API is disabled
		if ei.Reason == "SERVICE_DISABLED" && ei.Domain == "googleapis.com" {
			return true
		}
	}

	return false
}
