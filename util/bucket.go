package util

import "errors"

func FindBucket(config *Config, bucketName string) (Bucket, int, error) {
	for i, b := range config.Buckets {
		if b.Alias == bucketName {
			return b, i, nil
		}
	}
	return Bucket{}, -1, errors.New("Bucket not exist! Use \"./coscli config show\" to check config file please! ")
}
