package localize

type (
	ctxLangKey     struct{}
	ctxLocalizeKey struct{}
)

var (
	langKey      = ctxLangKey{}
	localizerKey = ctxLocalizeKey{}
)
