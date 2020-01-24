// Copyright © 2020 The Things Industries B.V.

package packetbrokeragent

import (
	"context"

	"go.thethings.network/lorawan-stack/pkg/tenant"
	"go.thethings.network/lorawan-stack/pkg/ttipb"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/pkg/types"
)

// DevAddrTenantIdentifier provides the tenant identifiers based on a DevAddr.
type DevAddrTenantIdentifier interface {
	TenantIDByDevAddr(context.Context, types.DevAddr) (ttipb.TenantIdentifiers, error)
}

// WithTenancyContextFiller returns an Option that fills the tenant context.
func WithTenancyContextFiller(id DevAddrTenantIdentifier) Option {
	return WithEndDeviceIdentifiersContextFiller(func(parent context.Context, ids ttnpb.EndDeviceIdentifiers) (context.Context, error) {
		if ids.DevAddr == nil {
			return parent, nil
		}
		tntID, err := id.TenantIDByDevAddr(parent, *ids.DevAddr)
		if err != nil {
			return nil, err
		}
		return tenant.NewContext(parent, tntID), nil
	})
}
