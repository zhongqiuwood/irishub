package gov

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"fmt"
)

// Handle all "gov" type messages.
func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case MsgDeposit:
			return handleMsgDeposit(ctx, keeper, msg)
		case MsgSubmitProposal:
			return handleMsgSubmitProposal(ctx, keeper, msg)
		case MsgVote:
			return handleMsgVote(ctx, keeper, msg)
		default:
			errMsg := "Unrecognized gov msg type"
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleMsgSubmitProposal(ctx sdk.Context, keeper Keeper, msg MsgSubmitProposal) sdk.Result {

	proposal := keeper.NewProposal(ctx, msg.Title, msg.Description, msg.ProposalType,msg.Params)

	err, votingStarted := keeper.AddDeposit(ctx, proposal.GetProposalID(), msg.Proposer, msg.InitialDeposit)
	if err != nil {
		return err.Result()
	}

	proposalIDBytes := keeper.cdc.MustMarshalBinaryBare(proposal.GetProposalID())

	tags := sdk.NewTags(
		"action", []byte("submitProposal"),
		"proposer", []byte(msg.Proposer.String()),
		"proposalId", proposalIDBytes,
	)

	if votingStarted {
		tags.AppendTag("votingPeriodStart", proposalIDBytes)
	}

	return sdk.Result{
		Data: proposalIDBytes,
		Tags: tags,
	}
}

func handleMsgDeposit(ctx sdk.Context, keeper Keeper, msg MsgDeposit) sdk.Result {

	err, votingStarted := keeper.AddDeposit(ctx, msg.ProposalID, msg.Depositer, msg.Amount)
	if err != nil {
		return err.Result()
	}

	proposalIDBytes := keeper.cdc.MustMarshalBinaryBare(msg.ProposalID)

	// TODO: Add tag for if voting period started
	tags := sdk.NewTags(
		"action", []byte("deposit"),
		"depositer", []byte(msg.Depositer.String()),
		"proposalId", proposalIDBytes,
	)

	if votingStarted {
		tags.AppendTag("votingPeriodStart", proposalIDBytes)
	}

	return sdk.Result{
		Tags: tags,
	}
}

func handleMsgVote(ctx sdk.Context, keeper Keeper, msg MsgVote) sdk.Result {

	err := keeper.AddVote(ctx, msg.ProposalID, msg.Voter, msg.Option)
	if err != nil {
		return err.Result()
	}

	proposalIDBytes := keeper.cdc.MustMarshalBinaryBare(msg.ProposalID)

	tags := sdk.NewTags(
		"action", []byte("vote"),
		"voter", []byte(msg.Voter.String()),
		"proposalId", proposalIDBytes,
	)
	return sdk.Result{
		Tags: tags,
	}
}

// Called every block, process inflation, update validator set
func EndBlocker(ctx sdk.Context, keeper Keeper) (tags sdk.Tags, nonVotingVals []sdk.AccAddress) {

	logger := ctx.Logger().With("module", "x/gov")

	tags = sdk.NewTags()

	// Delete proposals that haven't met minDeposit
	for shouldPopInactiveProposalQueue(ctx, keeper) {
		inactiveProposal := keeper.InactiveProposalQueuePop(ctx)
		if inactiveProposal.GetStatus() == StatusDepositPeriod {
			proposalIDBytes := keeper.cdc.MustMarshalBinaryBare(inactiveProposal.GetProposalID())
			keeper.RefundDeposits(ctx, inactiveProposal.GetProposalID())
			keeper.DeleteProposal(ctx, inactiveProposal)
			tags.AppendTag("action", []byte("proposalDropped"))
			tags.AppendTag("proposalId", proposalIDBytes)
			logger.Info("Proposal %d - \"%s\" - didn't mean minimum deposit (had only %s), deleted",
				inactiveProposal.GetProposalID(), inactiveProposal.GetTitle(), inactiveProposal.GetTotalDeposit())
		}
	}

	var passes bool

	// Check if earliest Active Proposal ended voting period yet
	for shouldPopActiveProposalQueue(ctx, keeper) {
		activeProposal := keeper.ActiveProposalQueuePop(ctx)

		if ctx.BlockHeight() >= activeProposal.GetVotingStartBlock()+keeper.GetVotingProcedure(ctx).VotingPeriod {
			passes, nonVotingVals = tally(ctx, keeper, activeProposal)
			proposalIDBytes := keeper.cdc.MustMarshalBinaryBare(activeProposal.GetProposalID())
			if passes {
				keeper.RefundDeposits(ctx, activeProposal.GetProposalID())
				activeProposal.SetStatus(StatusPassed)
				tags.AppendTag("action", []byte("proposalPassed"))
				tags.AppendTag("proposalId", proposalIDBytes)
				activeProposal.Execute(ctx,keeper)
			} else {

				keeper.RefundDeposits(ctx, activeProposal.GetProposalID())
				activeProposal.SetStatus(StatusRejected)
				tags.AppendTag("action", []byte("proposalRejected"))
				tags.AppendTag("proposalId", proposalIDBytes)
			}

			logger.Info("Proposal %d - \"%s\" - tallied, passed: %v",
				activeProposal.GetProposalID(), activeProposal.GetTitle(), passes)
			keeper.SetProposal(ctx, activeProposal)

			SortAddresses(nonVotingVals)
			for _, valAddr := range nonVotingVals {
				val := keeper.ds.GetValidatorSet().Validator(ctx, valAddr)
				keeper.ds.GetValidatorSet().Slash(ctx,
					val.GetPubKey(),
					ctx.BlockHeight(),
					val.GetPower().RoundInt64(),
					keeper.GetTallyingProcedure(ctx).GovernancePenalty)

				logger.Info(fmt.Sprintf("Validator %s failed to vote on proposal %d, slashing",
					val.GetOwner(), activeProposal.GetProposalID()))
			}
		}
	}

	return tags, nonVotingVals
}
func shouldPopInactiveProposalQueue(ctx sdk.Context, keeper Keeper) bool {
	depositProcedure := keeper.GetDepositProcedure(ctx)
	peekProposal := keeper.InactiveProposalQueuePeek(ctx)

	if peekProposal == nil {
		return false
	} else if peekProposal.GetStatus() != StatusDepositPeriod {
		return true
	} else if ctx.BlockHeight() >= peekProposal.GetSubmitBlock()+depositProcedure.MaxDepositPeriod {
		return true
	}
	return false
}

func shouldPopActiveProposalQueue(ctx sdk.Context, keeper Keeper) bool {
	votingProcedure := keeper.GetVotingProcedure(ctx)
	peekProposal := keeper.ActiveProposalQueuePeek(ctx)

	if peekProposal == nil {
		return false
	} else if ctx.BlockHeight() >= peekProposal.GetVotingStartBlock()+votingProcedure.VotingPeriod {
		return true
	}
	return false
}