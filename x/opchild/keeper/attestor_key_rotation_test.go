package keeper_test

import (
	"testing"

	testutilsims "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/OPinit/x/opchild/testutil"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func Test_HandleAttestorSetUpdatePacket_UpdatesConsensusPubkeyForExistingOperator(t *testing.T) {
	ctx, input := testutil.CreateTestInput(t, false)

	bridgeInfo := opchildtypes.BridgeInfo{
		BridgeId:   1,
		BridgeAddr: testutil.AddrsStr[0],
		L1ChainId:  "test-chain-1",
		L1ClientId: "test-client-id",
		BridgeConfig: ophosttypes.BridgeConfig{
			AttestorSet: []ophosttypes.Attestor{},
		},
	}
	require.NoError(t, input.OPChildKeeper.BridgeInfo.Set(ctx, bridgeInfo))

	pubKeys := testutilsims.CreateTestPubKeys(2)
	operatorAddress := testutil.ValAddrsStr[1]

	initialAttestor := testutil.CreateAttestor(t, operatorAddress, pubKeys[0], "attestor")
	initialAck, err := input.OPChildKeeper.HandleAttestorSetUpdatePacket(ctx, ophosttypes.AttestorSetUpdatePacketData{
		BridgeId:    1,
		AttestorSet: []ophosttypes.Attestor{initialAttestor},
	})
	require.NoError(t, err)
	require.True(t, initialAck.Success)

	// The complete L1 attestor-set replacement keeps the operator address
	// but rotates its consensus public key.
	rotatedAttestor := testutil.CreateAttestor(t, operatorAddress, pubKeys[1], "attestor")
	rotatedAck, err := input.OPChildKeeper.HandleAttestorSetUpdatePacket(ctx, ophosttypes.AttestorSetUpdatePacketData{
		BridgeId:    1,
		AttestorSet: []ophosttypes.Attestor{rotatedAttestor},
	})
	require.NoError(t, err)
	require.True(t, rotatedAck.Success)

	validators, err := input.OPChildKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	require.Len(t, validators, 1)

	consAddr, err := validators[0].GetConsAddr()
	require.NoError(t, err)
	require.Equal(t, sdk.ConsAddress(pubKeys[1].Address()), consAddr,
		"validator consensus key should match the newly registered L1 attestor set")
}
