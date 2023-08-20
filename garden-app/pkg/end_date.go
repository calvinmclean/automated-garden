package pkg

// EndDateable is a simple interface that requires a method to determine if something is end-dated
type EndDateable interface {
	EndDated() bool
}
