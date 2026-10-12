package routespec

type ParamIn string

const (
	InPath   ParamIn = "path"
	InQuery  ParamIn = "query"
	InHeader ParamIn = "header"
)

type Operation struct {
	Request     any
	ID          string
	Summary     string
	Description string
	Tags        []string
	Responses   []Response
	Params      []Param
	Security    []Security
	Deprecated  bool
}

type Response struct {
	Body        any
	ContentType string
	Description string
	Status      int
}

type Param struct {
	In          ParamIn
	Name        string
	Description string
	Required    bool
}

type Security struct {
	Name   string
	Scopes []string
}
