package main

import "fmt"

type LimitedQueue struct {
	MaxLen int
	Items  []interface{}
}

func (q *LimitedQueue) Append(item interface{}) {
	q.Items = append(q.Items, item)
	if len(q.Items) > q.MaxLen {
		q.Items = q.Items[1:]
	}
}

func (q *LimitedQueue) ToStringSlice() []string {
	var result []string
	for _, item := range q.Items {
		result = append(result, fmt.Sprintf("%v", item))
	}
	return result
}
