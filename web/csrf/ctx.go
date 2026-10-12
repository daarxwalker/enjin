package csrf

type (
	ctxKeyType struct{}
)

var ctxKey = ctxKeyType{}

type ctxVal struct {
	input    *InputManager
	rejected bool
}
