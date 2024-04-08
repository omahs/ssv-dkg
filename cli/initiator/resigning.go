package initiator

import (
	"context"
	"fmt"
	"log"

	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	cli_utils "github.com/bloxapp/ssv-dkg/cli/utils"
	"github.com/bloxapp/ssv-dkg/pkgs/initiator"
	"github.com/bloxapp/ssv-dkg/pkgs/wire"
)

func init() {
	cli_utils.SetResigningFlags(StartResigning)
}

var StartResigning = &cobra.Command{
	Use:   "resign",
	Short: "Resigning DKG results",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(`
		█████╗ ██╗  ██╗ ██████╗     ██╗███╗   ██╗██╗████████╗██╗ █████╗ ████████╗ ██████╗ ██████╗ 
		██╔══██╗██║ ██╔╝██╔════╝     ██║████╗  ██║██║╚══██╔══╝██║██╔══██╗╚══██╔══╝██╔═══██╗██╔══██╗
		██║  ██║█████╔╝ ██║  ███╗    ██║██╔██╗ ██║██║   ██║   ██║███████║   ██║   ██║   ██║██████╔╝
		██║  ██║██╔═██╗ ██║   ██║    ██║██║╚██╗██║██║   ██║   ██║██╔══██║   ██║   ██║   ██║██╔══██╗
		██████╔╝██║  ██╗╚██████╔╝    ██║██║ ╚████║██║   ██║   ██║██║  ██║   ██║   ╚██████╔╝██║  ██║
		╚═════╝ ╚═╝  ╚═╝ ╚═════╝     ╚═╝╚═╝  ╚═══╝╚═╝   ╚═╝   ╚═╝╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝`)
		if err := cli_utils.SetViperConfig(cmd); err != nil {
			return err
		}
		if err := cli_utils.BindResigningFlags(cmd); err != nil {
			return err
		}
		logger, err := cli_utils.SetGlobalLogger(cmd, "dkg-initiator")
		if err != nil {
			return err
		}
		defer func() {
			if err := cli_utils.Sync(logger); err != nil {
				log.Printf("Failed to sync logger: %v", err)
			}
		}()
		logger.Info("🪛 Initiator`s", zap.String("Version", cmd.Version))
		// Load operators
		opMap, err := cli_utils.LoadOperators(logger)
		if err != nil {
			logger.Fatal("😥 Failed to load operators: ", zap.Error(err))
		}
		operatorIDs, err := cli_utils.StingSliceToUintArray(cli_utils.OperatorIDs)
		if err != nil {
			logger.Fatal("😥 Failed to load participants: ", zap.Error(err))
		}
		allProofs, err := cli_utils.LoadProofs(cli_utils.ProofsFilePath)
		if err != nil {
			logger.Fatal("😥 Failed to read pro json file:", zap.Error(err))
		}
		// start the ceremony
		ctx := context.Background()
		pool := pool.NewWithResults[*Result]().WithContext(ctx).WithFirstError().WithMaxGoroutines(maxConcurrency)
		for i := 0; i < len(allProofs); i++ {
			i := i
			pool.Go(func(ctx context.Context) (*Result, error) {
				// Create new DKG initiator
				_, err := initiator.New(opMap.Clone(), logger, cmd.Version, cli_utils.ClientCACertPath)
				if err != nil {
					return nil, err
				}
				logger.Info("Loaded proofs", zap.Any("proof", &allProofs[i]), zap.Any("operators", operatorIDs))
				// Create a new ID.
				// id := crypto.NewID()
				// nonce := cli_utils.Nonce + uint64(i)
				// Perform the ceremony.
				// depositData, keyShares, proofs, err := dkgInitiator.StartDKG(id, cli_utils.WithdrawAddress.Bytes(), operatorIDs, ethnetwork, cli_utils.OwnerAddress, nonce)
				// if err != nil {
				// 	return nil, err
				// }
				// logger.Debug("DKG ceremony completed",
				// 	zap.String("id", hex.EncodeToString(id[:])),
				// 	zap.Uint64("nonce", nonce),
				// 	zap.String("pubkey", depositData.PubKey),
				// )
				return nil, nil
				// return &Result{
				// 	id:          id,
				// 	depositData: depositData,
				// 	keyShares:   keyShares,
				// 	nonce:       nonce,
				// 	proof:       proofs,
				// }, nil
			})
		}
		results, err := pool.Wait()
		if err != nil {
			logger.Fatal("😥 Failed to initiate DKG ceremony: ", zap.Error(err))
		}
		var keySharesArr []*wire.KeySharesCLI
		var proofs [][]*wire.SignedProof
		for _, res := range results {
			keySharesArr = append(keySharesArr, res.keyShares)
			proofs = append(proofs, res.proof)
		}
		// Save results
		logger.Info("🎯 All data is validated.")
		if err := cli_utils.WriteResults(
			logger,
			nil,
			keySharesArr,
			proofs,
			false,
			int(cli_utils.Validators),
			cli_utils.OwnerAddress,
			cli_utils.Nonce,
			cli_utils.WithdrawAddress,
			cli_utils.OutputPath,
		); err != nil {
			logger.Fatal("Could not save results", zap.Error(err))
		}
		logger.Info("🚀 DKG ceremony completed")
		return nil
	},
}
