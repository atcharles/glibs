package util

type Map map[string]interface{}

// ToBean ...
func (m Map) ToBean(v interface{}) error {
	val, err := Marshal(m)
	if err != nil {
		return err
	}
	return Unmarshal(val, v)
}

// Bean2Map ...
func Bean2Map(v interface{}) Map {
	val, _ := Marshal(v)
	m := make(Map)
	_ = Unmarshal(val, &m)
	return m
}

func MergeMap(target, source Map) {
	for k, v := range source {
		target[k] = v
	}
}

// MergeBean ...if source's field is not empty, then set target's field to source's field
// target and source must be pointer
// if source's field is empty, then target's field will not be changed
func MergeBean(target, source interface{}) error {
	t := Bean2Map(target)
	s := Bean2Map(source)
	MergeMap(t, s)
	return t.ToBean(target)
}
