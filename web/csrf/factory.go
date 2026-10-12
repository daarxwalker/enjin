package csrf

import (
	"context"

	"github.com/a-h/templ"
)

func NewInputFrom(ctx context.Context) templ.Component {
	cv := from(ctx)
	if cv == nil || cv.input == nil {
		return nil
	}
	return cv.input.CreateInput(ctx)
}
