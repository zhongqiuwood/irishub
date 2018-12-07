package keeper

import (
	sdk "github.com/irisnet/irishub/types"
	"github.com/irisnet/irishub/modules/distribution/types"
)

// Allocate fees handles distribution of the collected fees
func (k Keeper) AllocateTokens(ctx sdk.Context, percentVotes sdk.Dec, proposer sdk.ConsAddress) {

	// get the proposer of this block
	proposerValidator := k.stakeKeeper.ValidatorByConsAddr(ctx, proposer)

	proposerDist := k.GetValidatorDistInfo(ctx, proposerValidator.GetOperator())

	// get the fees which have been getting collected through all the
	// transactions in the block
	feesCollected := k.feeCollectionKeeper.GetCollectedFees(ctx)
	feesCollectedDec := types.NewDecCoins(feesCollected)

	// allocated rewards to proposer
	baseProposerReward := k.GetBaseProposerReward(ctx)
	bonusProposerReward := k.GetBonusProposerReward(ctx)
	proposerMultiplier := baseProposerReward.Add(bonusProposerReward.Mul(percentVotes))
	proposerReward := feesCollectedDec.MulDec(proposerMultiplier)

	// apply commission
	commission := proposerReward.MulDec(proposerValidator.GetCommission())
	remaining := proposerReward.Minus(commission)
	proposerDist.ValCommission = proposerDist.ValCommission.Plus(commission)
	proposerDist.DelPool = proposerDist.DelPool.Plus(remaining)

	// allocate community funding
	communityTax := k.GetCommunityTax(ctx)
	communityFunding := feesCollectedDec.MulDec(communityTax)
	feePool := k.GetFeePool(ctx)
	feePool.CommunityPool = feePool.CommunityPool.Plus(communityFunding)

	// set the global pool within the distribution module
	poolReceived := feesCollectedDec.Minus(proposerReward).Minus(communityFunding)
	feePool.ValPool = feePool.ValPool.Plus(poolReceived)

	k.SetValidatorDistInfo(ctx, proposerDist)
	k.SetFeePool(ctx, feePool)

	// clear the now distributed fees
	k.feeCollectionKeeper.ClearCollectedFees(ctx)
}

// Allocate fee tax handles distribution of the community fee pool
func (k Keeper) AllocateFeeTax(ctx sdk.Context, destAddr sdk.AccAddress, percent sdk.Dec, burn bool) {
	feePool := k.GetFeePool(ctx)
	communityPool := feePool.CommunityPool
	allocateCoins, decCoins := communityPool.MulDec(percent).TruncateDecimal()
	feePool.CommunityPool = communityPool.Minus(types.NewDecCoins(allocateCoins).Plus(decCoins))
	k.SetFeePool(ctx, feePool)

	if burn {
		stakeDenom := k.stakeKeeper.GetStakeDenom(ctx)
		for _, coin := range allocateCoins {
			if coin.Denom == stakeDenom {
				k.stakeKeeper.BurnTokens(ctx, sdk.NewDecFromInt(coin.Amount))
			}
		}
		return
	}

	k.bankKeeper.AddCoins(ctx, destAddr, allocateCoins)

}
