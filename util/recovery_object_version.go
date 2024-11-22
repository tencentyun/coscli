package util

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// RecoveryObjectVersion recovery object version
func RecoveryObjectVersion(c *cos.Client, cosUrl StorageUrl, previous int, limit int, dryrun bool, anyVersion bool) error {
	var total = 0
	var marker = ""
	var versionMarker = ""
	var isTruncated = true

	var ctx = context.Background()
	var prefix = cosUrl.(*CosUrl).Object

	for isTruncated && total < limit {
		var opt = &cos.BucketGetObjectVersionsOptions{
			Prefix:          prefix,
			KeyMarker:       marker,
			VersionIdMarker: versionMarker,
		}
		resp, _, err := c.Bucket.GetObjectVersions(ctx, opt)
		if err != nil {
			return err
		}

		total += len(resp.Version)
		total += len(resp.DeleteMarker)

		var notLatestVersions = versionKeyMap(false, resp.Version)
		var log = logrus.WithField("versions", notLatestVersions).
			WithField("previous", previous).
			WithField("dryrun", dryrun)

		for key, versions := range notLatestVersions {
			logrus.WithField("Key", key).WithField("versions", versions).Info("find versions")
		}

		if anyVersion {
			for key, versions := range notLatestVersions {
				if len(versions) < previous || previous <= 0 {
					log.WithField("previous", previous).Warn("version not found")
					return nil
				}
				err = recoveryObject(c, key, versions[previous-1].VersionId, dryrun)
				if err != nil {
					log.WithError(err).Warn("recovery failed")
				} else {
					log.Info("recovery success")
				}
			}
		} else {
			for _, marker := range resp.DeleteMarker {
				var log = log.WithField("Key", marker.Key).
					WithField("marker.VersionId", marker.VersionId)

				if marker.IsLatest {
					versionIds, has := notLatestVersions[marker.Key]
					if has && (len(versionIds) < previous || previous <= 0) {
						log.Warn("version not found")
						return nil
					}

					if has {
						err = recoveryObject(c, marker.Key, versionIds[previous-1].VersionId, dryrun)
						if err != nil {
							log.WithError(err).Warn("recovery failed")
						} else {
							log.Info("recovery success")
						}
					} else {
						log.Warn("not found the latest version of delete marker")
					}
				}
			}
		}

		logrus.WithField("IsTruncated", resp.IsTruncated).WithField("NextKeyMarker", resp.NextKeyMarker).Info("DeleteMarkers")
		marker = resp.NextKeyMarker
		isTruncated = resp.IsTruncated
		versionMarker = resp.NextVersionIdMarker

		if !isTruncated || total >= limit {
			logrus.Info("no more")
			break
		}
	}
	return nil
}

// map[Key]Version
func versionKeyMap(latest bool, versions []cos.ListVersionsResultVersion) map[string][]cos.ListVersionsResultVersion {
	var versionMap = make(map[string][]cos.ListVersionsResultVersion)
	for _, version := range versions {
		if version.IsLatest == latest {
			versionMap[version.Key] = append(versionMap[version.Key], version)
		} else {
		}
	}
	return versionMap
}

func recoveryObject(c *cos.Client, key string, versionId string, dryrun bool) (err error) {
	var destKey = key
	if versionId == "" {
		return nil
	}
	var srcUrl = fmt.Sprintf("%s/%s", c.BaseURL.BucketURL.Host, key)

	var log = logrus.WithField("srcUrl", srcUrl).WithField("VersionId", versionId).WithField("DestKey", destKey)
	if dryrun {
		log.Info("DryRun RecoveryObject")
		return nil
	}

	resp, _, err := c.Object.Copy(context.Background(), key, srcUrl, nil, versionId)

	var respVerId string
	if resp != nil {
		respVerId = resp.VersionId
	}

	log.WithError(err).
		WithField("respVerId", respVerId).
		Warn("RecoveryObject")

	return err
}
