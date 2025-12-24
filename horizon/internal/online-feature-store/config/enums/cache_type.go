package enums

type CacheType string

const (
	CacheTypeDistributed CacheType = "distributed"
	CacheTypeInMemory    CacheType = "in-memory"
)

func (c CacheType) String() string {
	return string(c)
}
