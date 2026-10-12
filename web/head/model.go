package head

type Meta struct {
	Name    string
	Content string
}

type Link struct {
	Href string
	Type string
	Rel  string
}

type Script struct {
	Src   string
	Defer bool
}
