package repository

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

// Helper functions shared across all test files

func int32Ptr(i int32) *int32 { return &i }

func boolPtr(b bool) *bool { return &b }

func int64Ptr(i int64) *int64 { return &i }

func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(err)
	}
	return q
}

func resourceQuantityPtr(s string) *resource.Quantity {
	q := resource.MustParse(s)
	return &q
}
