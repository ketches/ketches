package builder

type Builder interface {
	Build(option interface{}) (BuildResult, error)
}

type BuildResult struct {
	BuilderID string
}
