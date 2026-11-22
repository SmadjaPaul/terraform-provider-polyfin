package polyfin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// VersionCriteria mirrors upstream accounts.VersionCriteria (Polyfin 0.85):
// the values Settings > Playback > Version order holds.
var VersionCriteria = []string{"language", "resolution-high", "resolution-low", "size-large", "size-small", "size-per-minute", "codec",
	"hdr-first", "hdr-last", "source", "tier", "audio", "edition", "labels", "cached", "seeders"}

// MaxVersionCriteria mirrors upstream accounts.MaxVersionCriteria.
const MaxVersionCriteria = 6

func versionCriterionKind(criterion string) string {
	kind, _, _ := strings.Cut(criterion, "-")
	return kind
}

// ValidateVersionOrder applies upstream's rule: known criteria, one of each
// kind (resolution-high and resolution-low are one kind), six at most. An
// empty order keeps the addons' order.
func ValidateVersionOrder(order []string) error {
	if len(order) > MaxVersionCriteria {
		return errors.New("version order holds at most 6 criteria")
	}
	for i, criterion := range order {
		if !slices.Contains(VersionCriteria, criterion) {
			return fmt.Errorf("unknown version criterion %q", criterion)
		}
		if slices.ContainsFunc(order[:i], func(before string) bool {
			return versionCriterionKind(before) == versionCriterionKind(criterion)
		}) {
			return fmt.Errorf("version criterion %q repeats a kind already in the order", criterion)
		}
	}
	return nil
}

// GetVersionOrder reads the criteria ordering a title's versions; the
// "language" criterion ranks the user's preferred audio language first.
func (c *Client) GetVersionOrder(ctx context.Context) ([]string, error) {
	if err := c.CheckAdmin(ctx); err != nil {
		return nil, err
	}
	var result struct {
		VersionOrder *[]string `json:"versionOrder"`
	}
	if _, err := c.request(ctx, "GET", "/admin/api/settings", nil, &result); err != nil {
		return nil, err
	}
	if result.VersionOrder == nil {
		return nil, errors.New("Polyfin settings missing versionOrder")
	}
	return *result.VersionOrder, nil
}

// SetVersionOrder sends only versionOrder (partial PUT) and reads it back.
// The codec, source and edition sub-orders are not touched.
func (c *Client) SetVersionOrder(ctx context.Context, order []string) error {
	if err := ValidateVersionOrder(order); err != nil {
		return err
	}
	if err := c.CheckAdmin(ctx); err != nil {
		return err
	}
	if order == nil {
		order = []string{}
	}
	if _, err := c.request(ctx, "PUT", "/admin/api/settings", map[string][]string{"versionOrder": order}, nil); err != nil {
		return err
	}
	current, err := c.GetVersionOrder(ctx)
	if err != nil {
		return err
	}
	if !slices.Equal(current, order) {
		return fmt.Errorf("Polyfin versionOrder readback mismatch")
	}
	return nil
}
