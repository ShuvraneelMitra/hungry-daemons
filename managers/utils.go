package managers

import "sort"

type LineageCount struct {
    Surname   string
    Count     int
}

type SortedMap struct {
    data     map[string]int
    Lineages []LineageCount
}

func (sm *SortedMap) Set(key string, value int) {
    if _, exists := sm.data[key]; exists {
        sm.Delete(key)
    }
    sm.data[key] = value
    // Binary search for insertion point
    i := sort.Search(len(sm.Lineages), func(i int) bool {
        return sm.Lineages[i].Count >= value
    })
    sm.Lineages = append(sm.Lineages, LineageCount{})
    copy(sm.Lineages[i+1:], sm.Lineages[i:])
    sm.Lineages[i] = LineageCount{key, value}
}

func (sm *SortedMap) Delete(key string) {
    val := sm.data[key]
    delete(sm.data, key)
    i := sort.Search(len(sm.Lineages), func(i int) bool {
        return sm.Lineages[i].Count >= val
    })
	// for duplicates
    for i < len(sm.Lineages) && sm.Lineages[i].Surname != key {
        i++
    }
    sm.Lineages = append(sm.Lineages[:i], sm.Lineages[i+1:]...)
}

func (sm *SortedMap) Get(key string) (int, bool) {
    v, ok := sm.data[key]
    return v, ok
}

func (sm *SortedMap) TopK(k int, descending bool) []LineageCount {
    if len(sm.Lineages) < k {
        return sm.Lineages
    }
    
	result := make([]LineageCount, k)
	if descending {
		copy(result, sm.Lineages[len(sm.Lineages) - k : ])		
	} else {
		copy(result, sm.Lineages[ : k])
	}
	return result
}
