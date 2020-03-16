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
	"context"
	"runtime/trace"

	"github.com/go-redis/redis"
	"go.thethings.network/lorawan-stack/pkg/errors"
	"go.thethings.network/lorawan-stack/pkg/gatewayserver/io"
	ttnredis "go.thethings.network/lorawan-stack/pkg/redis"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/pkg/unique"
)

// GatewayConnectionStatsRegistry implements the GatewayConnectionStatsRegistry interface.
type GatewayConnectionStatsRegistry struct {
	Redis *ttnredis.Client
}

var (
	down        = "down"
	up          = "up"
	status      = "status"
	errNotFound = errors.DefineNotFound("stats_not_found", "Gateway Stats not found")
)

func (r *GatewayConnectionStatsRegistry) key(which string, uid string) string {
	return r.Redis.Key(which, "uid", uid)
}

// Set sets or clears the connection stats for a gateway.
func (r *GatewayConnectionStatsRegistry) Set(ctx context.Context, ids ttnpb.GatewayIdentifiers, stats *ttnpb.GatewayConnectionStats, update io.Traffic) error {
	uid := unique.ID(ctx, ids)

	defer trace.StartRegion(ctx, "set gateway connection stats").End()

	var err error

	if stats == nil {
		// Delete if nil
		err = r.Redis.Del(r.key(up, uid), r.key(down, uid), r.key(status, uid)).Err()
	} else {
		// Update (pipelined for better performance) otherwise
		_, err = r.Redis.Pipelined(func(p redis.Pipeliner) error {
			if update.Up {
				if _, err = ttnredis.SetProto(p, r.key(up, uid), stats, 0); err != nil {
					return err
				}
			}
			if update.Down {
				if _, err = ttnredis.SetProto(p, r.key(down, uid), stats, 0); err != nil {
					return err
				}
			}
			if update.Status {
				if _, err = ttnredis.SetProto(p, r.key(status, uid), stats, 0); err != nil {
					return err
				}
			}
			if !update.Up && !update.Down && !update.Status {
				if _, err = ttnredis.SetProto(p, r.key(status, uid), stats, 0); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err != nil {
		return ttnredis.ConvertError(err)
	}
	return nil
}

// Get returns the connection stats for a gateway.
func (r *GatewayConnectionStatsRegistry) Get(ctx context.Context, ids ttnpb.GatewayIdentifiers) (*ttnpb.GatewayConnectionStats, error) {
	uid := unique.ID(ctx, ids)
	result := &ttnpb.GatewayConnectionStats{}
	stats := &ttnpb.GatewayConnectionStats{}

	retrieved, err := r.Redis.MGet(r.key(up, uid), r.key(down, uid), r.key(status, uid)).Result()
	if err != nil {
		return nil, ttnredis.ConvertError(err)
	}

	if retrieved[0] == nil && retrieved[1] == nil && retrieved[2] == nil {
		return nil, errNotFound
	}

	// uplink stats
	if retrieved[0] != nil {
		if ttnredis.UnmarshalProto(retrieved[0].(string), stats) == nil {
			result.LastUplinkReceivedAt = stats.LastUplinkReceivedAt
			result.UplinkCount = stats.UplinkCount
			result.RoundTripTimes = stats.RoundTripTimes
		} else {
			return nil, errNotFound
		}
	}

	// downlink stats
	if retrieved[1] != nil {
		if ttnredis.UnmarshalProto(retrieved[1].(string), stats) == nil {
			result.LastDownlinkReceivedAt = stats.LastDownlinkReceivedAt
			result.DownlinkCount = stats.DownlinkCount
		} else {
			return nil, errNotFound
		}
	}

	// gateway status
	if retrieved[2] != nil {
		if ttnredis.UnmarshalProto(retrieved[2].(string), stats) == nil {
			result.ConnectedAt = stats.ConnectedAt
			result.Protocol = stats.Protocol
			result.LastStatus = stats.LastStatus
			result.LastStatusReceivedAt = stats.LastStatusReceivedAt
		} else {
			return nil, errNotFound
		}
	}

	return result, nil
}
