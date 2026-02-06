package httpclient

// IHttpStatusHandler is an interface for handling HTTP request statuses
type IHttpStatusHandler interface {
	// OnRequest handles a request with its status result
	OnRequest(status string)
	// OnRetry handles retry events
	OnRetry()
}
