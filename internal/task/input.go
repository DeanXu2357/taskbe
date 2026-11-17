package task

const DefaultPageSize = 20

type CreateInput struct {
	Name   string
	Status Status
}

type ReplaceInput struct {
	Name   string
	Status Status
}

type ListParams struct {
	AfterID int64
	Limit   int
}

type Page struct {
	Tasks   []Task
	HasMore bool
}
