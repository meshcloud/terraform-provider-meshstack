package poll

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// AtMostFor is a factory for Pollable and enables inference of T by compiler from given func return value *T.
// Optionally supports catching the last result with WithLastResultTo.
func AtMostFor[T any](timeout time.Duration, f Func[T], options ...Option[T]) (pollable Pollable[T]) {
	pollable.f = f
	pollable.timeout = timeout
	for _, option := range options {
		option(&pollable)
	}
	return
}

type Func[T any] func(ctx context.Context) (*T, error)

type Option[T any] func(*Pollable[T])

func WithLastResultTo[T any](resultTarget **T) Option[T] {
	return func(pollable *Pollable[T]) {
		pollable.resultOutput = resultTarget
	}
}

// Pollable is created with convenient factories such as AtMostFor.
type Pollable[T any] struct {
	// f is called during polling in Pollable.Until.
	f Func[T]
	// timeout must be set along with Func.
	timeout time.Duration
	// resultOutput is optional and can thus be nil.
	// If not nil, is used to update the latest result of Func.
	// See Pollable.Until.
	resultOutput **T
}

// Until retries Pollable.Func until the given predicate indicates done.
func (pollable Pollable[T]) Until(ctx context.Context, until func(item *T) (done bool, err error)) error {
	return retry.RetryContext(ctx, pollable.timeout, func() *retry.RetryError {
		item, err := pollable.f(ctx)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("retrieving item failed: %w", err))
		}
		if pollable.resultOutput != nil {
			*pollable.resultOutput = item
		}
		if done, err := until(item); err != nil {
			return retry.NonRetryableError(fmt.Errorf("item in failed state: %w", err))
		} else if done {
			return nil
		} else {
			return retry.RetryableError(fmt.Errorf("item %T not (yet) in desired state", item))
		}
	})
}
