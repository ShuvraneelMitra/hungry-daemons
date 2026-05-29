package managers

import "sort"

type LineageCount struct {
    Surname   string
    Count     int
}

type SortedMap struct {
    data     map[string]int
    lineages []LineageCount
}

func (sm *SortedMap) Set(key string, value int) {
    if _, exists := sm.data[key]; exists {
        sm.Delete(key)
    }
    sm.data[key] = value
    // Binary search for insertion point
    i := sort.Search(len(sm.lineages), func(i int) bool {
        return sm.lineages[i].Count >= value
    })
    sm.lineages = append(sm.lineages, LineageCount{})
    copy(sm.lineages[i+1:], sm.lineages[i:])
    sm.lineages[i] = LineageCount{key, value}
}

func (sm *SortedMap) Delete(key string) {
    val := sm.data[key]
    delete(sm.data, key)
    i := sort.Search(len(sm.lineages), func(i int) bool {
        return sm.lineages[i].Count >= val
    })
	// for duplicates
    for i < len(sm.lineages) && sm.lineages[i].Surname != key {
        i++
    }
    sm.lineages = append(sm.lineages[:i], sm.lineages[i+1:]...)
}

func (sm *SortedMap) Get(key string) (int, bool) {
    v, ok := sm.data[key]
    return v, ok
}

func (sm *SortedMap) TopK(k int, descending bool) []LineageCount {
    if len(sm.lineages) < k {
        return sm.lineages
    }
    
	result := make([]LineageCount, k)
	if descending {
		copy(result, sm.lineages[len(sm.lineages) - k : ])		
	} else {
		copy(result, sm.lineages[ : k])
	}
	return result
}
