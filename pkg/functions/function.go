package functions

type Function struct {
	Name string
	Runtime string
	Memory int
}

func GetFunction (name string) (*Function, bool) {
	//TODO: info should be retrieved from the DB (or possibly through a
	//local cache)
	return &Function{"prova", "python38", 256}, true
}

func (f *Function) String() string {
	return f.Name
}
