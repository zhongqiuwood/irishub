package cli

import (
	flag "github.com/spf13/pflag"
)

const (
	FlagReceiver             = "receiver"
	FlagReceiverOnOtherChain = "receiver-on-other-chain"
	FlagHashLock             = "hash-lock"
	FlagInAmount             = "in-amount"
	FlagAmount               = "amount"
	FlagTimeLock             = "time-lock"
	FlagTimestamp            = "timestamp"
	FlagSecret               = "secret"
)

var (
	FsCreateHTLC = flag.NewFlagSet("", flag.ContinueOnError)
	FsClaimHTLC  = flag.NewFlagSet("", flag.ContinueOnError)
	FsRefundHTLC = flag.NewFlagSet("", flag.ContinueOnError)
)

func init() {
	FsCreateHTLC.String(FlagReceiver, "", "Bech32 encoding address to receive coins")
	FsCreateHTLC.BytesHex(FlagReceiverOnOtherChain, nil, "the receiver address on the other chain")
	FsCreateHTLC.BytesHex(FlagHashLock, nil, "the sha256 hash generated by secret and timestamp")
	FsCreateHTLC.Uint64(FlagTimestamp, 0, "the timestamp for generating hashLock, accurate to the second")
	FsCreateHTLC.Uint64(FlagInAmount, 0, "the expected gained token on the other chain")
	FsCreateHTLC.String(FlagAmount, "", "similar to the amount in the original transfer")
	FsCreateHTLC.String(FlagTimeLock, "", "the number of blocks to wait before the asset may be returned to")

	FsClaimHTLC.BytesHex(FlagHashLock, nil, "the secret for generating hashLock")
	FsClaimHTLC.String(FlagSecret, "", "the secret for generating hashLock")

	FsRefundHTLC.BytesHex(FlagHashLock, nil, "the secret for generating hashLock")
}
