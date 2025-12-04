package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/idna"

	"github.com/hexiosec/asm-cloud-connector/internal/api"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	asm "github.com/hexiosec/asm-sdk-go"
)

const (
	resourceDomain string = "Domain"
	resourceIPv4   string = "IPv4"
	resourceIPv6   string = "IPv6"
)

type Connector struct {
	scanID      string
	seedTag     string
	deleteStale bool
	sdk         api.API
}

func NewConnector(cfg *config.Config, sdk api.API) (*Connector, error) {
	return &Connector{
		scanID:      cfg.ScanID,
		seedTag:     cfg.SeedTag,
		deleteStale: cfg.DeleteStaleSeeds,
		sdk:         sdk,
	}, nil
}

// Checks you can authenticate with the API key and the scan exists
func (c *Connector) Authenticate(ctx context.Context) error {
	resp, _, err := c.sdk.GetState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth state, %w", err)
	}

	if !resp.Authenticated {
		return fmt.Errorf("credentials not valid")
	}

	_, _, err = c.sdk.GetScanByID(ctx, c.scanID)
	if err != nil {
		return fmt.Errorf("failed to check %s scan exists, %w", c.scanID, err)
	}

	return nil
}

// SyncResources synchronises local resources with ASM seeds.
// Returns an error only for fatal conditions (e.g. API unavailable).
// Known validation failures or best-effort deletions are logged and skipped.
func (c *Connector) SyncResources(ctx context.Context, resources []string) error {
	// Remove duplicates
	resources = dedup(ctx, resources)

	// Normalise i.e. extract domains from websites
	resources = normalise(ctx, resources)

	// Remove duplicates again - just in case
	resources = dedup(ctx, resources)

	// Get existing seeds
	existingSeeds, err := c.getSeeds(ctx)
	if err != nil {
		return err
	}

	// Add seeds to scan, if they don't exist
	for _, resource := range resources {
		iCtx := logger.WithLogger(ctx, logger.GetLogger(ctx).With().Str("resource", resource).Logger())
		logger.GetLogger(iCtx).Trace().Msg("Processing resource")

		if _, ok := existingSeeds[resource]; ok {
			delete(existingSeeds, resource)
			logger.GetLogger(iCtx).Debug().Msgf("Seed %s already exists", resource)
			continue
		}

		resourceType := getResourceType(resource)
		if resourceType == resourceIPv6 {
			logger.GetLogger(iCtx).Warn().Msg("Cannot add IPv6 as seed, skipping")
			continue
		}

		logger.GetLogger(iCtx).Debug().Msgf("Adding seed %s", resource)
		// Semgrep false positive: resp is nil-checked before use
		// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
		_, resp, err := c.sdk.AddScanSeedById(
			ctx,
			c.scanID,
			asm.CreateScanSeedRequest{
				Name: resource,
				Type: resourceType,
				Tags: []string{c.seedTag},
			},
		)
		if err != nil {
			// Attempt to classify known recoverable errors (e.g. invalid seed, already exists)
			if resp != nil && resp.StatusCode == http.StatusBadRequest && resp.Body != nil {
				code, rErr := getErrorCode(resp.Body)
				if rErr != nil {
					logger.GetLogger(iCtx).Error().Err(rErr).Msg("failed to get error code from response to determine why the seed couldn't be added")
					// This may indicate a deeper API issue -> abort
					return fmt.Errorf("failed to add seed %s %w", resource, err)
				}

				// Known non-fatal case: seed invalid skip and continue.
				logger.GetLogger(iCtx).Warn().Err(err).Str("code", code).Msgf("failed to add seed %s because %s", resource, code)
				continue
			}

			// Unexpected failure -> abort
			return fmt.Errorf("failed to add seed %s %w", resource, err)
		}
	}

	if !c.deleteStale {
		// Nothing more to do
		logger.GetLogger(ctx).Trace().Msg("Not deleting stale seeds")
		return nil
	}

	logger.GetLogger(ctx).Trace().Msg("Deleting stale seeds")
	// Deletion is best-effort: log but don't abort
	// Stale seeds are existingSeeds that aren't in the resource list and have a matching seed tag, implying it was previously added by the Cloud Connector
	for _, seed := range existingSeeds {
		if !slices.Contains(seed.Tags, c.seedTag) {
			logger.GetLogger(ctx).Debug().Msgf("skipping existing seed %s as it doesn't have tag %s, so was probably added manually", seed.Name, c.seedTag)
			continue
		}

		logger.GetLogger(ctx).Debug().Str("seed", seed.Name).Msgf("Removing seed %s", seed.Name)
		_, err := c.sdk.RemoveScanSeedById(ctx, c.scanID, seed.Id)
		if err != nil {
			logger.GetLogger(ctx).Error().Err(err).Msgf("failed to remove stale seed %s", seed.Name)
		}
	}

	return nil
}

func (c *Connector) getSeeds(ctx context.Context) (map[string]*asm.SeedsResponseInner, error) {
	seeds, _, err := c.sdk.GetScanSeedsById(ctx, c.scanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan %s existing seeds %w", c.scanID, err)
	}

	byName := make(map[string]*asm.SeedsResponseInner, len(seeds))
	for _, seed := range seeds {
		s := seed
		byName[seed.Name] = &s
	}

	return byName, nil
}

func dedup(ctx context.Context, resources []string) []string {
	if len(resources) < 2 {
		return resources
	}

	seen := make(map[string]struct{}, len(resources))
	writeIdx := 0
	for _, res := range resources {
		if _, exists := seen[res]; exists {
			continue
		}
		seen[res] = struct{}{}
		resources[writeIdx] = res
		writeIdx++
	}

	logger.GetLogger(ctx).Trace().Msgf("Removed %d duplicates", len(resources)-writeIdx)

	return resources[:writeIdx]
}

func normalise(ctx context.Context, resources []string) []string {
	if len(resources) == 0 {
		return nil
	}

	log := logger.GetLogger(ctx)
	normalised := make([]string, 0, len(resources))

	for _, raw := range resources {
		value, ok := normaliseResource(raw)
		if !ok {
			log.Warn().Str("resource", raw).Msg("Unable to normalise resource")
			continue
		}

		normalised = append(normalised, value)
		if raw != value {
			log.Debug().Str("resource", raw).Str("normalised", value).Msgf("'%s' was normalised to '%s'", raw, value)
		}

	}

	return normalised
}

func normaliseResource(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	// Strip leading wildcard prefix (e.g. *.example.com) before further parsing
	raw = strings.TrimPrefix(raw, "*.")

	// handle bare IPv6
	if strings.Contains(raw, ":") && !strings.Contains(raw, "[") {
		if ip := net.ParseIP(raw); ip != nil && ip.To4() == nil {
			raw = "[" + raw + "]"
		}
	}

	// Ensure it has a scheme so url.Parse behaves consistently
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", false
	}

	host := u.Hostname()

	// IPv6, IPv4, or domain
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), true
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if len(host) == 0 {
		return "", false
	}

	// Validate as FQDN
	if _, err := idna.Lookup.ToASCII(host); err != nil {
		return "", false
	}

	return host, true
}

func getResourceType(resource string) string {
	ip := net.ParseIP(resource)
	if ip == nil {
		// Not an IP, assume domain
		return resourceDomain
	}
	if ip.To4() != nil {
		return resourceIPv4
	}
	return resourceIPv6
}

func getErrorCode(body io.ReadCloser) (string, error) {
	defer body.Close()
	errBody := struct {
		Code string `json:"code"`
	}{}
	err := json.NewDecoder(body).Decode(&errBody)
	if err != nil {
		return "", err
	}

	if errBody.Code == "" {
		return "", fmt.Errorf("no code")
	}

	return errBody.Code, nil
}
