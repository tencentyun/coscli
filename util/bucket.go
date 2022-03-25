package util

func FindBucket(config *Config, bucketName string) (Bucket, int, error) {
	for i, b := range config.Buckets {
		if b.Alias == bucketName {
			return b, i, nil
		}
	}
	for i, b := range config.Buckets {
		if b.Name == bucketName {
			return b, i, nil
		}
	}
	var tmpBucket Bucket
	tmpBucket.Name = bucketName
	return tmpBucket, -1, nil
	// return Bucket{}, -1, errors.New("Bucket not exist! Use \"./coscli config show\" to check config file please! ")
}
