package scheduling

type Policy interface {
	Init()
	OnCompletion(request *scheduledRequest)
	OnArrival(request *scheduledRequest)
	OnRestore(restore *scheduledRestore)
}
