package cli

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/wire"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/irisnet/irishub/client/context"
	"github.com/irisnet/irishub/modules/upgrade"
	"github.com/irisnet/irishub/modules/upgrade/params"
	"github.com/spf13/cobra"
	"os"
)

func GetInfoCmd(storeName string, cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "query the information of upgrade module",
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithLogger(os.Stdout).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			res_height, _ := cliCtx.QueryStore([]byte("gov/"+upgradeparams.ProposalAcceptHeightParameter.GetStoreKey()), "params")
			res_proposalID, _ := cliCtx.QueryStore([]byte("gov/"+upgradeparams.CurrentUpgradeProposalIdParameter.GetStoreKey()), "params")
			var height int64
			var proposalID int64
			cdc.MustUnmarshalBinary(res_height, &height)
			cdc.MustUnmarshalBinary(res_proposalID, &proposalID)

			res_versionID, _ := cliCtx.QueryStore(upgrade.GetCurrentVersionKey(), storeName)
			var versionID int64
			cdc.MustUnmarshalBinary(res_versionID, &versionID)

			res_version, _ := cliCtx.QueryStore(upgrade.GetVersionIDKey(versionID), storeName)
			var version upgrade.Version
			cdc.MustUnmarshalBinary(res_version, &version)
			output, err := wire.MarshalJSONIndent(cdc, version)
			if err != nil {
				return err
			}

			fmt.Println(string(output))
			fmt.Println("CurrentProposalId           = ", proposalID)
			fmt.Println("CurrentProposalAcceptHeight = ", height)
			return nil
		},
	}
	return cmd
}
