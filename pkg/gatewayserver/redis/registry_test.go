// Copyright © 2020 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redis

import (
	"testing"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/smartystreets/assertions"
	"go.thethings.network/lorawan-stack/pkg/errors"
	"go.thethings.network/lorawan-stack/pkg/gatewayserver/io"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/pkg/types"
	"go.thethings.network/lorawan-stack/pkg/util/test"
	"go.thethings.network/lorawan-stack/pkg/util/test/assertions/should"
)

var (
	Timeout    = 10 * test.Delay
	allTraffic = io.Traffic{Up: true, Down: true, Status: true}
)

func TestRegistry(t *testing.T) {
	a := assertions.New(t)

	ctx := test.Context()

	cl, flush := test.NewRedis(t, "redis_test")
	defer flush()
	defer cl.Close()

	ids := ttnpb.GatewayIdentifiers{
		GatewayID: "gtw1",
		EUI:       &types.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	ids2 := ttnpb.GatewayIdentifiers{
		GatewayID: "gtw2",
		EUI:       &types.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	ids3 := ttnpb.GatewayIdentifiers{
		GatewayID: "gtw3",
		EUI:       &types.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
	}
	registry := &GatewayConnectionStatsRegistry{
		Redis: cl,
	}

	now := time.Now().UTC()
	initialStats := &ttnpb.GatewayConnectionStats{
		ConnectedAt:            &now,
		Protocol:               "dummy",
		LastDownlinkReceivedAt: &now,
		DownlinkCount:          1,
		LastUplinkReceivedAt:   &now,
		UplinkCount:            1,
		LastStatusReceivedAt:   nil,
		LastStatus:             nil,
	}

	emptyStats := &ttnpb.GatewayConnectionStats{
		ConnectedAt:          &now,
		Protocol:             "dummy",
		LastStatusReceivedAt: nil,
		LastStatus:           nil,
	}

	t.Run("GetNonExisting", func(t *testing.T) {
		stats, err := registry.Get(ctx, ids)
		a.So(stats, should.BeNil)
		a.So(errors.IsNotFound(err), should.BeTrue)
	})

	t.Run("EmptyStats", func(t *testing.T) {
		err := registry.Set(ctx, ids3, emptyStats, io.Traffic{})
		a.So(err, should.BeNil)
		retrieved, err := registry.Get(ctx, ids3)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, emptyStats)
	})

	t.Run("SetAndClear", func(t *testing.T) {
		err := registry.Set(ctx, ids, initialStats, allTraffic)
		a.So(err, should.BeNil)
		retrieved, err := registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, initialStats)

		// Other gateways not affected
		stats, err := registry.Get(ctx, ids2)
		a.So(stats, should.BeNil)
		a.So(errors.IsNotFound(err), should.BeTrue)

		// Unset
		err = registry.Set(ctx, ids, nil, io.Traffic{})
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(errors.IsNotFound(err), should.BeTrue)
		a.So(retrieved, should.BeNil)

	})

	t.Run("ClearManyTimes", func(t *testing.T) {
		a.So(registry.Set(ctx, ids, nil, io.Traffic{}), should.BeNil)
		a.So(registry.Set(ctx, ids, nil, io.Traffic{}), should.BeNil)
	})

	t.Run("UpdateUplink", func(t *testing.T) {
		now := time.Now().UTC().Add(time.Minute)
		stats := deepcopy.Copy(initialStats).(*ttnpb.GatewayConnectionStats)

		// Update uplink stats, make sure they work
		stats.UplinkCount = 10
		stats.LastUplinkReceivedAt = &now
		err := registry.Set(ctx, ids, stats, allTraffic)
		a.So(err, should.BeNil)
		retrieved, err := registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, stats)

		// Keep a copy of the uplink stats
		correct := deepcopy.Copy(stats)

		// Update downlink stats as well, expect no change
		stats.DownlinkCount += 100
		err = registry.Set(ctx, ids, stats, io.Traffic{Up: true})
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, correct)

		// Now update downlink also
		stats.LastDownlinkReceivedAt = &now
		err = registry.Set(ctx, ids, stats, io.Traffic{Up: true, Down: true})
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, stats)
	})

	t.Run("UpdateDownlink", func(t *testing.T) {
		now := time.Now().UTC().Add(2 * time.Minute)
		stats := deepcopy.Copy(initialStats).(*ttnpb.GatewayConnectionStats)

		// Reset stats from previous test
		a.So(registry.Set(ctx, ids, nil, io.Traffic{}), should.BeNil)
		a.So(registry.Set(ctx, ids, stats, allTraffic), should.BeNil)
		retrieved, err := registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, stats)

		// Update downlink stats, make sure they work
		stats.DownlinkCount = 10
		stats.LastDownlinkReceivedAt = &now
		err = registry.Set(ctx, ids, stats, io.Traffic{Down: true})
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, stats)

		// Keep a copy of the dowlink stats
		correct := deepcopy.Copy(stats)

		// Update uplink stats as well, expect no change
		stats.UplinkCount += 100
		err = registry.Set(ctx, ids, stats, io.Traffic{Down: true})
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, correct)

		// Now update uplink also
		stats.LastUplinkReceivedAt = &now
		err = registry.Set(ctx, ids, stats, io.Traffic{Down: true, Up: true})
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, stats)
	})

	t.Run("UpdateStats", func(t *testing.T) {
		now := time.Now().UTC().Add(3 * time.Minute)
		stats := deepcopy.Copy(initialStats).(*ttnpb.GatewayConnectionStats)

		// Reset stats from previous test
		a.So(registry.Set(ctx, ids, nil, io.Traffic{}), should.BeNil)
		a.So(registry.Set(ctx, ids, stats, allTraffic), should.BeNil)
		retrieved, err := registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, stats)

		// Update status
		stats.LastStatusReceivedAt = &now
		stats.LastStatus = &ttnpb.GatewayStatus{
			IP:   []string{"10.10.10.10"},
			Time: now,
			Metrics: map[string]float32{
				"a": 3.22,
				"b": 3.42,
			},
		}

		// Keep correct stats
		correct := deepcopy.Copy(stats).(*ttnpb.GatewayConnectionStats)

		// Mess with the uplink and downlink stats
		stats.UplinkCount += 100
		stats.DownlinkCount += 1000

		// Update gateway status only
		err = registry.Set(ctx, ids, stats, io.Traffic{Status: true})
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, correct)

		// Now update uplink and downlink
		stats.LastUplinkReceivedAt = &now
		stats.LastDownlinkReceivedAt = &now
		err = registry.Set(ctx, ids, stats, allTraffic)
		a.So(err, should.BeNil)
		retrieved, err = registry.Get(ctx, ids)
		a.So(err, should.BeNil)
		a.So(retrieved, should.Resemble, stats)
	})
}
