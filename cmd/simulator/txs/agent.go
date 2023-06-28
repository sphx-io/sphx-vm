// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// TxSequence provides an interface to return a channel of transactions.
// The sequence is responsible for closing the channel when there are no further
// transactions.
type TxSequence[T any] interface {
	Chan() <-chan T
}

// Worker defines the interface for issuance and confirmation of transactions.
// The caller is responsible for calling Close to cleanup resources used by the
// worker at the end of the simulation.
type Worker[T any] interface {
	IssueTx(ctx context.Context, tx T) error
	ConfirmTx(ctx context.Context, tx T) error
	Close(ctx context.Context) error
}

// Execute the work of the given agent.
type Agent[T any] interface {
	Execute(ctx context.Context) error
}

// issueNAgent issues and confirms a batch of N transactions at a time.
type issueNAgent[T any] struct {
	sequence TxSequence[T]
	worker   Worker[T]
	n        uint64
}

// NewIssueNAgent creates a new issueNAgent
func NewIssueNAgent[T any](sequence TxSequence[T], worker Worker[T], n uint64) Agent[T] {
	return &issueNAgent[T]{
		sequence: sequence,
		worker:   worker,
		n:        n,
	}
}

// Execute issues txs in batches of N and waits for them to confirm
func (a issueNAgent[T]) Execute(ctx context.Context) error {
	if a.n == 0 {
		return errors.New("batch size n cannot be equal to 0")
	}

	txChan := a.sequence.Chan()
	totalTxs := len(txChan)
	confirmedCount := 0
	start := time.Now()

	for {
		var (
			txs = make([]T, 0, a.n)
			tx  T
			ok  bool
		)
		for i := uint64(0); i < a.n; i++ {
			select {
			case tx, ok = <-txChan:
				if !ok {
					return a.worker.Close(ctx)
				}
			case <-ctx.Done():
				return ctx.Err()
			}

			if err := a.worker.IssueTx(ctx, tx); err != nil {
				return fmt.Errorf("failed to issue transaction %d: %w", len(txs), err)
			}
			txs = append(txs, tx)
		}

		for i, tx := range txs {
			if err := a.worker.ConfirmTx(ctx, tx); err != nil {
				return fmt.Errorf("failed to await transaction %d: %w", i, err)
			}
			confirmedCount++
			if confirmedCount == totalTxs {
				totalTime := time.Since(start).Seconds()
				log.Info("Execution complete", "totalTxs", totalTxs, "totalTime", totalTime, "TPS", float64(totalTxs)/totalTime)
				logOtherMetrics()
			}
		}
	}
}

func logOtherMetrics() error {
	resp, err := http.Get("http://127.0.0.1:9650/ext/metrics")
	if err != nil {
		return fmt.Errorf("failed getting metrics: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed reading response body of metrics: %w", err)
	}

	bodyString := string(body)
	re := regexp.MustCompile(".*avalanche_C_vm_metervm_build_block_sum.*")
	matches := re.FindAllStringSubmatch(bodyString, -1)

	log.Info("Metrics", "results", matches[len(matches)-1])
	return nil
}
