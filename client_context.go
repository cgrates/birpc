package birpc

import "context"

type clientKey struct{}

// WithClient returns a context containing client.
func WithClient(ctx context.Context, client ClientConnector) context.Context {
	return context.WithValue(ctx, clientKey{}, client)
}

// ClientFromContext returns the client stored in ctx, or nil.
func ClientFromContext(ctx context.Context) ClientConnector {
	client, _ := ctx.Value(clientKey{}).(ClientConnector)
	return client
}
