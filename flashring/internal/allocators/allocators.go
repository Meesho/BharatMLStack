package allocators

type SizeClass struct {
	Size     int
	MinCount int
}

type Meta struct {
	Size int
	Name string
}
