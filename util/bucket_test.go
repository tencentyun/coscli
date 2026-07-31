package util

import (
	"testing"
)

func TestFindBucket(t *testing.T) {
	config := &Config{
		Buckets: []Bucket{
			{Name: "bucket-a-1234567890", Alias: "alias-a", Region: "ap-guangzhou"},
			{Name: "bucket-b-1234567890", Alias: "alias-b", Region: "ap-beijing"},
			{Name: "bucket-c-1234567890", Alias: "", Region: "ap-shanghai"},
		},
	}

	t.Run("按 alias 查找成功", func(t *testing.T) {
		bucket, idx, err := FindBucket(config, "alias-a")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if idx != 0 {
			t.Errorf("期望 idx=0，实际 idx=%d", idx)
		}
		if bucket.Name != "bucket-a-1234567890" {
			t.Errorf("期望 Name=bucket-a-1234567890，实际 Name=%s", bucket.Name)
		}
		if bucket.Region != "ap-guangzhou" {
			t.Errorf("期望 Region=ap-guangzhou，实际 Region=%s", bucket.Region)
		}
	})

	t.Run("按 name 查找成功", func(t *testing.T) {
		bucket, idx, err := FindBucket(config, "bucket-b-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if idx != 1 {
			t.Errorf("期望 idx=1，实际 idx=%d", idx)
		}
		if bucket.Alias != "alias-b" {
			t.Errorf("期望 Alias=alias-b，实际 Alias=%s", bucket.Alias)
		}
	})

	t.Run("按 name 查找（无 alias 的桶）", func(t *testing.T) {
		bucket, idx, err := FindBucket(config, "bucket-c-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if idx != 2 {
			t.Errorf("期望 idx=2，实际 idx=%d", idx)
		}
		if bucket.Name != "bucket-c-1234567890" {
			t.Errorf("期望 Name=bucket-c-1234567890，实际 Name=%s", bucket.Name)
		}
	})

	t.Run("未找到时返回临时 bucket（idx=-1）", func(t *testing.T) {
		bucket, idx, err := FindBucket(config, "not-exist-bucket")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if idx != -1 {
			t.Errorf("期望 idx=-1，实际 idx=%d", idx)
		}
		if bucket.Name != "not-exist-bucket" {
			t.Errorf("期望临时 bucket Name=not-exist-bucket，实际 Name=%s", bucket.Name)
		}
	})

	t.Run("空配置时返回临时 bucket", func(t *testing.T) {
		emptyConfig := &Config{}
		bucket, idx, err := FindBucket(emptyConfig, "any-bucket")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if idx != -1 {
			t.Errorf("期望 idx=-1，实际 idx=%d", idx)
		}
		if bucket.Name != "any-bucket" {
			t.Errorf("期望 Name=any-bucket，实际 Name=%s", bucket.Name)
		}
	})
}
