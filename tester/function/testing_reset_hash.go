package function

import (
	"github.com/ysugimoto/falco/interpreter/context"
	"github.com/ysugimoto/falco/interpreter/function/errors"
	"github.com/ysugimoto/falco/interpreter/value"
)

const Testing_reset_hash_Name = "testing.reset_hash"

func Testing_reset_hash(
	ctx *context.Context,
	args ...value.Value,
) (value.Value, error) {
	if len(args) != 0 {
		return nil, errors.NewTestingError(
			"%s: expects no arguments, got %d", Testing_reset_hash_Name, len(args),
		)
	}
	ctx.RequestHash.Value = ""
	return value.Null, nil
}
