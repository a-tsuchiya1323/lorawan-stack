// Copyright © 2020 The Things Industries B.V.

package cluster

import "go.thethings.network/lorawan-stack/pkg/ttipb"

// PacketBrokerTenantID is the proxy tenant identifier of requests made through Packet Broker.
var PacketBrokerTenantID = ttipb.TenantIdentifiers{TenantID: "packetbroker"}
