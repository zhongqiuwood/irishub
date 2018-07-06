package consensus

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/cosmos/cosmos-sdk/wire" // XXX fix
	"fmt"
	"strings"
	"encoding/hex"
	"time"
	"log"
	cctx "context"
)

// Metrics contains metrics exposed by this package.
type Metrics struct {
	// Height of the chain.
	Height metrics.Gauge

	// Number of rounds.
	Rounds metrics.Gauge

	// Number of validators.
	Validators metrics.Gauge
	// Total power of all validators.
	ValidatorsPower metrics.Gauge
	// Number of validators who did not sign.
	MissingValidators metrics.Gauge
	// Total power of the missing validators.
	MissingValidatorsPower metrics.Gauge
	// Number of validators who tried to double sign.
	ByzantineValidators metrics.Gauge
	// Total power of the byzantine validators.
	ByzantineValidatorsPower metrics.Gauge

	// Time between this and the last block.
	BlockIntervalSeconds metrics.Histogram

	// Number of transactions.
	NumTxs metrics.Gauge
	// Size of the block.
	BlockSizeBytes metrics.Gauge
	// Total number of transactions.
	TotalTxs metrics.Gauge
}

// PrometheusMetrics returns Metrics build using Prometheus client library.
func PrometheusMetrics() *Metrics {
	return &Metrics{
		Height: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "height",
			Help:      "Height of the chain.",
		}, []string{}),
		Rounds: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "rounds",
			Help:      "Number of rounds.",
		}, []string{}),

		Validators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "validators",
			Help:      "Number of validators.",
		}, []string{}),
		ValidatorsPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "validators_power",
			Help:      "Total power of all validators.",
		}, []string{}),
		MissingValidators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "missing_validators",
			Help:      "Number of validators who did not sign.",
		}, []string{}),
		MissingValidatorsPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "missing_validators_power",
			Help:      "Total power of the missing validators.",
		}, []string{}),
		ByzantineValidators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "byzantine_validators",
			Help:      "Number of validators who tried to double sign.",
		}, []string{}),
		ByzantineValidatorsPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "byzantine_validators_power",
			Help:      "Total power of the byzantine validators.",
		}, []string{}),

		BlockIntervalSeconds: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Subsystem: "consensus",
			Name:      "block_interval_seconds",
			Help:      "Time between this and the last block.",
			Buckets:   []float64{1, 2.5, 5, 10, 60},
		}, []string{}),

		NumTxs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "num_txs",
			Help:      "Number of transactions.",
		}, []string{}),
		BlockSizeBytes: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "block_size_bytes",
			Help:      "Size of the block.",
		}, []string{}),
		TotalTxs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "consensus",
			Name:      "total_txs",
			Help:      "Total number of transactions.",
		}, []string{}),
	}
}


func Monitor(ctx context.CoreContext,csMetrics Metrics,cdc *wire.Codec,storeName string){
	context, _ := cctx.WithTimeout(cctx.Background(), 10*time.Second)

	var client = ctx.Client
	//开启监听事件
	client.Start()

	blockC := make(chan interface{})

	err := client.Subscribe(context, "monitor", types.EventQueryNewBlock, blockC)

	if err != nil {
		log.Println("got ", err)
	}

	go func() {
		for e := range blockC {
			block := e.(types.TMEventData).(types.EventDataNewBlock)
			csMetrics.RecordMetrics(ctx,cdc,block.Block,storeName)
		}
	}()
}

func (cs Metrics) RecordMetrics(ctx context.CoreContext,cdc *wire.Codec,block *types.Block,storeName string) {
	cs.Height.Set(float64(block.Height))
	cs.ByzantineValidators.Set(float64(len(block.Evidence.Evidence)))

	missingValidators := 0
	missingValidatorsPower := int64(0)
	validators := getValidators(cdc,ctx,storeName)

	valMap := make(map[string]stake.Validator,len(validators))
	for i, val := range validators {
		var vote *types.Vote
		if i < len(block.LastCommit.Precommits) {
			vote = block.LastCommit.Precommits[i]
		}
		if vote == nil {
			missingValidators++
			missingValidatorsPower += val.GetPower().Evaluate()
		}

		valMap[val.Owner.String()] = val
	}

	cs.MissingValidators.Set(float64(missingValidators))
	cs.MissingValidatorsPower.Set(float64(missingValidatorsPower))

	byzantineValidatorsPower := int64(0)
	for _, ev := range block.Evidence.Evidence {
		addr := strings.ToUpper(hex.EncodeToString(ev.Address()))
		if val,ok := valMap[addr]; ok {
			byzantineValidatorsPower += val.GetPower().Evaluate()
		}
	}
	cs.ByzantineValidatorsPower.Set(float64(byzantineValidatorsPower))

	if block.Height > 1 {
		lastBlockHight := block.Height -1
		resultBlock,_ := ctx.Client.Block(&lastBlockHight)
		cs.BlockIntervalSeconds.Observe(
			block.Time.Sub(resultBlock.BlockMeta.Header.Time).Seconds(),
		)
	}

	cs.NumTxs.Set(float64(block.NumTxs))
	cs.TotalTxs.Set(float64(block.TotalTxs))

	bz, _ := cdc.MarshalBinaryBare(block)
	cs.BlockSizeBytes.Set(float64(len(bz)))
}


func getValidators(cdc *wire.Codec,ctx context.CoreContext,storeName string) (validators []stake.Validator){
	key := stake.ValidatorsKey
	resKVs, err := ctx.QuerySubspace(cdc, key, storeName)
	if err != nil {
		fmt.Println(err)
	}

	for _, KV := range resKVs {
		var validator stake.Validator
		cdc.MustUnmarshalBinary(KV.Value, &validator)
		validators = append(validators, validator)
	}
	return validators
}