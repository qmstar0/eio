package message

var setCtxKey = &contextKey{"setCtxKey"}

type contextKey struct {
	message string
}

func (k *contextKey) String() string {
	return "context value " + k.message
}
