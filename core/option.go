package gkit

// Option sets an optional parameter for handlers.
type Option[Handler any] func(Handler)
