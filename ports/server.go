package ports

type PublicServer interface {
	Serve() error
}
