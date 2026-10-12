package abuse

type Policy struct {
	Name   string
	Rules  []Rule
	DryRun bool
}
