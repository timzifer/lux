package ui

// sendFunc is the function used to inject messages into the app loop.
// Set by the framework during integration (see SetSendFunc).
var sendFunc func(any)

// SetSendFunc sets the function used to inject messages into the app loop.
// Called by app.runInternal during initialization. Passing nil disconnects
// the hook; Send becomes a no-op.
func SetSendFunc(fn func(any)) { sendFunc = fn }

// Send injects a message into the app loop. It is a no-op when no send
// function is wired (e.g. in isolated widget tests).
func Send(msg any) {
	if sendFunc != nil {
		sendFunc(msg)
	}
}
