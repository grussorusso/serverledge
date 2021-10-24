package functions

type Function struct {
	Name string
	Runtime string // example: python310
	Memory int
	Handler string // example: "module.function_name"
	SourceTarURL string
}

func GetFunction (name string) (*Function, bool) {
	//TODO: info should be retrieved from the DB (or possibly through a
	//local cache)
	return &Function{"prova", "python310", 256, "function.handler", "/home/gabriele/function.tar"}, true
}

func (f *Function) String() string {
	return f.Name
}
