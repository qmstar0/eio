package message

type Header map[string]string

func (m Header) Get(key string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return ""
}

func (m Header) Set(key, value string) {
	m[key] = value
}
