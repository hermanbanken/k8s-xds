package internal

import (
	"math/rand"
	"reflect"
	"sort"
)

// prioritySort pulls the preferred argument to the front of the slice
func prioritySort(str interface{}, preferred string) {
	sort.Slice(str, func(i, j int) bool {
		return reflect.ValueOf(str).Index(i).Interface() == preferred
	})
}

// Iterates over a slice, from a random offset
// for behavior like slice is a hashring.
func randomForEach(slice interface{}, r *rand.Rand, fn func(i int)) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		panic("invalid use of randomForEach: pass a slice")
	}
	if v.Len() == 0 {
		return
	}

	start := r.Intn(v.Len())
	for i := start; i < v.Len(); i++ {
		fn(i)
	}
	for i := 0; i < start; i++ {
		fn(i)
	}
}
