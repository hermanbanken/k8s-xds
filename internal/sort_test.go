package internal

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomForEach(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	expected := []int{5, 6, 7, 8, 9, 0, 1, 2, 3, 4}
	actual := []int{}
	randomForEach(make([]string, 10), r, func(i int) {
		actual = append(actual, i)
	})
	assert.Equal(t, actual, expected)
}

func TestPrioritySort(t *testing.T) {
	elems := []string{"a", "b", "c", "d"}
	prioritySort(elems, "c")
	assert.Equal(t, []string{"c", "a", "b", "d"}, elems)

	elems = []string{"d", "c", "b", "a"}
	prioritySort(elems, "c")
	assert.Equal(t, []string{"c", "d", "b", "a"}, elems)
}
