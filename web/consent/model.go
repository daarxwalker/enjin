package consent

type Cookies struct {
	Analytic   bool
	Marketing  bool
	Preference bool
}

const (
	Analytic   = "analytic"
	Marketing  = "marketing"
	Preference = "preference"
)
