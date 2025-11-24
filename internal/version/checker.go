package version

import (
	"context"
	"fmt"
	h "net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/hexiosec/asm-cloud-connector/internal/http"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/hexiosec/asm-cloud-connector/internal/util"
)

var (
	// Injected at build time with the git tag
	// -ldflags "-X github.com/hexiosec/asm-cloud-connector/internal/version.version=$$(git describe --tags --abbrev=0)"
	version string = "0.0.0"
)

const (
	home string = "https://api.github.com/repos/hexiosec/asm-cloud-connector/releases/latest"
)

type checker struct {
	http http.IHttpService
}

func NewChecker(http http.IHttpService) (*checker, error) {
	return &checker{
		http: http,
	}, nil
}

type release struct {
	TagName string `mapstructure:"tag_name" validate:"required"`
}

// LogVersion compares the embedded build version to the latest Git tag and logs version status.
func (c *checker) LogVersion(ctx context.Context) {
	iCtx := logger.WithLogger(ctx, logger.GetLogger(ctx).With().Str("current", version).Logger())
	ok, remoteV, err := c.getLatestVersion(iCtx)
	if err != nil {
		logger.GetLogger(iCtx).Warn().Err(err).Msg("Failed to get latest version from remote repository")
		return
	}

	if !ok {
		logger.GetLogger(iCtx).Info().Msg("Request successful but no version found on remote repository, assuming latest version")
		return
	}

	newAvail, err := isGreaterThan(remoteV, version)
	if err != nil {
		logger.GetLogger(iCtx).Warn().Err(err).Str("remote", remoteV).Msg("Failed to compare current and remote version")
		return
	}

	if newAvail {
		logger.GetLogger(iCtx).Warn().Str("remote", remoteV).Msgf("New version available, %s", remoteV)
	} else {
		logger.GetLogger(iCtx).Info().Msgf("Running latest version, %s", version)
	}
}

func (c *checker) getLatestVersion(ctx context.Context) (bool, string, error) {
	resp, err := c.http.Get(ctx, home, http.HttpOptions{})
	if err != nil {
		return false, "", err
	}

	if resp.GetStatusCode() == h.StatusNotFound {
		// No release found
		return false, "", nil
	}

	if resp.GetStatusCode() != h.StatusOK {
		return false, "", fmt.Errorf("checker: received non-200 code %d", resp.GetStatusCode())
	}

	if !resp.HasBody() {
		return false, "", fmt.Errorf("checker: request successful but no body returned")
	}

	rel := release{}
	err = util.MapStructDecodeAndValidate(resp.GetBody(), &rel)
	if err != nil {
		return false, "", fmt.Errorf("checker: failed to destruct and validate response %w", err)
	}

	return true, rel.TagName, nil
}

// compares SemVer strings and returns a > b
func isGreaterThan(a, b string) (bool, error) {
	aCut, _ := strings.CutPrefix(a, "v")
	bCut, _ := strings.CutPrefix(b, "v")

	aVer, err := semver.NewVersion(aCut)
	if err != nil {
		return false, fmt.Errorf("checker: failed to parse a (%s) %w", a, err)
	}

	bVer, err := semver.NewVersion(bCut)
	if err != nil {
		return false, fmt.Errorf("checker: failed to parse b (%s) %w", b, err)
	}

	return aVer.GreaterThan(bVer), nil
}
