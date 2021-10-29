package functions

//A serverless Function.
type Function struct {
	Name         string
	Runtime      string // example: python310
	Memory       int
	Handler      string // example: "module.function_name"
	SourceTarURL string
}

//GetFunction retrieves a Function given its name.
func GetFunction(name string) (*Function, bool) {
	//TODO: info should be retrieved from the DB (or possibly through a
	//local cache)
	return &Function{"prova", "python310", 256, "function.handler", "http://www.ce.uniroma2.it/~russorusso/python310.tar"}, true
}

func (f *Function) String() string {
	return f.Name
}
