package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindBucket(t *testing.T) {
	config := &Config{
		Buckets: []Bucket{
			{Name: "name",
				Alias: "alias"},
		},
	}
	got, _, _ := FindBucket(config, "alias")
	want := "alias"
	assert.Equal(t, got.Alias, want, "they should be equal")
	got, _, _ = FindBucket(config, "name")
	want = "name"
	assert.Equal(t, got.Name, want, "they should be equal")
	got, _, _ = FindBucket(config, "invalid")
	want = "invalid"
	assert.Equal(t, got.Name, want, "they should be equal")
}
