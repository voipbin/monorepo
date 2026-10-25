package providerhandler

import (
	"context"
	"fmt"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/numberhandlertwilio"
)

// twilioProvider adapts numberhandlertwilio to Provider. The twilio handler implements none
// of the operations yet, so every operation reports provider.ErrNotImplemented. When twilio
// gets implemented, return the error of the handler as it is instead of wrapping it here.
type twilioProvider struct {
	h numberhandlertwilio.NumberHandlerTwilio
}

func (p *twilioProvider) GetAvailableNumbers(countryCode string, limit uint) ([]*availablenumber.AvailableNumber, error) {
	res, err := p.h.GetAvailableNumbers(countryCode, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", provider.ErrNotImplemented, err)
	}
	return res, nil
}

func (p *twilioProvider) NumberPurchase(num string) (*providernumber.ProviderNumber, error) {
	return nil, fmt.Errorf("%w: NumberPurchase", provider.ErrNotImplemented)
}

func (p *twilioProvider) NumberRelease(ctx context.Context, num *number.Number) error {
	if err := p.h.ReleaseNumber(ctx, num); err != nil {
		return fmt.Errorf("%w: %w", provider.ErrNotImplemented, err)
	}
	return nil
}

func (p *twilioProvider) NumberUpdateTags(ctx context.Context, num *number.Number, tags []string) error {
	return fmt.Errorf("%w: NumberUpdateTags", provider.ErrNotImplemented)
}
