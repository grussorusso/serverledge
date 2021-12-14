package api

type FunctionCreationRequest struct {
	Name            string
	Runtime         string
	Memory          int
	SourceTarBase64 string
	Handler         string
}
