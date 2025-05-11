package httptarget

type HttpTarget struct {
	Uri     string
	Headers map[string]string
}

// NewHttpTarget creates a new HttpTarget
func NewHttpTarget(uri string, headers map[string]string) HttpTarget {
	return HttpTarget{
		Uri:     uri,
		Headers: headers,
	}
}

// DispatchContents makes a POST request to the target Uri with the data
//
// Headers configured on the HttpTarget will be attached to the request
func (t HttpTarget) DispatchContents(data []byte) error {
	return nil
}
