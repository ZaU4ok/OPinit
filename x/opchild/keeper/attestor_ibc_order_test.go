package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	"github.com/initia-labs/OPinit/x/opchild"
	"github.com/initia-labs/OPinit/x/opchild/testutil"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func Test_IBCModule_OnRecvPacket_AcceptsOutOfOrderAttestorSetUpdate(t *testing.T) {
	ctx, input := testutil.CreateTestInput(t, false)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	const (
		l1ClientID      = "test-client-id"
		connectionID    = "connection-0"
		sourceChannel   = "channel-0"
		destinationChan = "channel-1"
	)

	bridgeInfo := opchildtypes.BridgeInfo{
		BridgeId:   1,
		BridgeAddr: testutil.AddrsStr[0],
		L1ChainId:  "test-chain-1",
		L1ClientId: l1ClientID,
		BridgeConfig: ophosttypes.BridgeConfig{
			AttestorSet: []ophosttypes.Attestor{},
			ChannelId:   sourceChannel,
		},
	}
	require.NoError(t, input.OPChildKeeper.BridgeInfo.Set(ctx, bridgeInfo))

	// OPChild channels are UNORDERED; deliver a newer update before an older one.
	input.IBCKeeper.ConnectionKeeper.SetClientConnectionPaths(sdkCtx, l1ClientID, []string{connectionID})
	input.IBCKeeper.ConnectionKeeper.SetConnection(sdkCtx, connectionID, connectiontypes.ConnectionEnd{
		State:    connectiontypes.OPEN,
		ClientId: l1ClientID,
	})
	input.IBCKeeper.ChannelKeeper.SetChannel(sdkCtx, opchildtypes.PortID, destinationChan, channeltypes.Channel{
		State: channeltypes.OPEN,
		Counterparty: channeltypes.Counterparty{
			PortId:    opchildtypes.PortID,
			ChannelId: sourceChannel,
		},
		ConnectionHops: []string{connectionID},
	})

	attestor1 := testutil.CreateAttestor(t, testutil.ValAddrsStr[1], testutil.PubKeys[1], "attestor1")
	attestor2 := testutil.CreateAttestor(t, testutil.ValAddrsStr[2], testutil.PubKeys[2], "attestor2")
	attestor3 := testutil.CreateAttestor(t, testutil.ValAddrsStr[3], testutil.PubKeys[3], "attestor3")

	newerPacketData := ophosttypes.NewAttestorSetUpdatePacketData(1, []ophosttypes.Attestor{attestor1, attestor3}, 200)
	stalePacketData := ophosttypes.NewAttestorSetUpdatePacketData(1, []ophosttypes.Attestor{attestor1, attestor2}, 100)

	newPacket := channeltypes.Packet{
		SourcePort:         opchildtypes.PortID,
		SourceChannel:      sourceChannel,
		DestinationPort:    opchildtypes.PortID,
		DestinationChannel: destinationChan,
		Data:               newerPacketData.GetBytes(),
		Sequence:           2,
	}
	stalePacket := channeltypes.Packet{
		SourcePort:         opchildtypes.PortID,
		SourceChannel:      sourceChannel,
		DestinationPort:    opchildtypes.PortID,
		DestinationChannel: destinationChan,
		Data:               stalePacketData.GetBytes(),
		Sequence:           1,
	}

	ibcModule := opchild.NewIBCModule(input.OPChildKeeper)
	newAck := ibcModule.OnRecvPacket(sdkCtx, opchildtypes.Version, newPacket, nil)
	require.True(t, newAck.Success(), "newer attestor-set packet should be accepted")

	staleAck := ibcModule.OnRecvPacket(sdkCtx, opchildtypes.Version, stalePacket, nil)
	require.True(t, staleAck.Success(), "current handler accepts the stale packet")

	updatedBridgeInfo, err := input.OPChildKeeper.BridgeInfo.Get(ctx)
	require.NoError(t, err)
	storedAttestors := make([]string, 0, len(updatedBridgeInfo.BridgeConfig.AttestorSet))
	for _, attestor := range updatedBridgeInfo.BridgeConfig.AttestorSet {
		storedAttestors = append(storedAttestors, attestor.Moniker)
	}
	require.ElementsMatch(t, []string{"attestor1", "attestor2"}, storedAttestors)

	validators, err := input.OPChildKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	activeAttestors := make([]string, 0, len(validators))
	for _, validator := range validators {
		if validator.ConsPower == int64(opchildtypes.AttestorConsPower) {
			activeAttestors = append(activeAttestors, validator.Moniker)
		}
	}
	require.ElementsMatch(t, []string{"attestor1", "attestor2"}, activeAttestors)
}
