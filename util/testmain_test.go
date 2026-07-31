package util

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// mockHttpClientDoFunc 全局 mock 变量，控制 http.Client.Do 行为
// （http.Client.Do 已在 TestMain 中全局打桩）
var mockHttpClientDoFunc func(req *http.Request) (*http.Response, error)

// TestMain 在所有测试运行前做一次全局打桩，避免 ARM64 上多次打桩同一方法的问题
func TestMain(m *testing.M) {
	var o *cos.ObjectService
	var b *cos.BucketService
	var svc *cos.ServiceService
	var httpClient *http.Client

	// 全局打桩 http.Client.Do（被 TestCamAuth、TestPutRename 等共用）
	patchesHttpClientDo := ApplyMethodFunc(reflect.TypeOf(httpClient), "Do",
		func(req *http.Request) (*http.Response, error) {
			if mockHttpClientDoFunc != nil {
				return mockHttpClientDoFunc(req)
			}
			return nil, nil
		})
	defer patchesHttpClientDo.Reset()

	// 全局打桩 Object.Head（被 TestStatObject 和 TestCheckCosObjectExist 共用）
	patchesHead := ApplyMethodFunc(reflect.TypeOf(o), "Head",
		func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			if mockHeadFunc != nil {
				return mockHeadFunc(ctx, name, opt, id...)
			}
			return nil, nil
		})
	defer patchesHead.Reset()

	// 全局打桩 Bucket.Get（被 TestCheckCosPathType 共用）
	patchesBucketGet := ApplyMethodFunc(reflect.TypeOf(b), "Get",
		func(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
			if mockBucketGetFunc != nil {
				return mockBucketGetFunc(ctx, opt)
			}
			return nil, nil, nil
		})
	defer patchesBucketGet.Reset()

	// 全局打桩 Bucket.Delete（被 TestRemoveBucket 共用）
	patchesBucketDelete := ApplyMethodFunc(reflect.TypeOf(b), "Delete",
		func(ctx context.Context, opt ...*cos.BucketDeleteOptions) (*cos.Response, error) {
			if mockBucketDeleteFunc != nil {
				return mockBucketDeleteFunc(ctx)
			}
			return nil, nil
		})
	defer patchesBucketDelete.Reset()

	// 全局打桩 Object.Delete（被 TestRemoveObjectOrVersion 共用）
	patchesObjectDelete := ApplyMethodFunc(reflect.TypeOf(o), "Delete",
		func(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
			if mockObjectDeleteFunc != nil {
				return mockObjectDeleteFunc(ctx, name, opt...)
			}
			return nil, nil
		})
	defer patchesObjectDelete.Reset()

	// 全局打桩 Object.DeleteMulti（被 TestDeleteCosObjects 和 TestDeleteCosObjectVersions 共用）
	patchesObjectDeleteMulti := ApplyMethodFunc(reflect.TypeOf(o), "DeleteMulti",
		func(ctx context.Context, opt *cos.ObjectDeleteMultiOptions) (*cos.ObjectDeleteMultiResult, *cos.Response, error) {
			if mockDeleteMultiFunc != nil {
				return mockDeleteMultiFunc(ctx, opt)
			}
			return nil, nil, nil
		})
	defer patchesObjectDeleteMulti.Reset()

	// 全局打桩 Bucket.GetObjectVersions（被 TestListObjectVersions 共用）
	patchesBucketGetObjectVersions := ApplyMethodFunc(reflect.TypeOf(b), "GetObjectVersions",
		func(ctx context.Context, opt *cos.BucketGetObjectVersionsOptions) (*cos.BucketGetObjectVersionsResult, *cos.Response, error) {
			if mockBucketGetObjectVersionsFunc != nil {
				return mockBucketGetObjectVersionsFunc(ctx, opt)
			}
			return nil, nil, nil
		})
	defer patchesBucketGetObjectVersions.Reset()

	// 全局打桩 Bucket.GetACL（被 TestGetBucketAcl 共用）
	patchesBucketGetACL := ApplyMethodFunc(reflect.TypeOf(b), "GetACL",
		func(ctx context.Context) (*cos.BucketGetACLResult, *cos.Response, error) {
			if mockBucketGetACLFunc != nil {
				return mockBucketGetACLFunc(ctx)
			}
			return nil, nil, nil
		})
	defer patchesBucketGetACL.Reset()

	// 全局打桩 Bucket.PutACL（被 TestPutBucketAcl 共用）
	patchesBucketPutACL := ApplyMethodFunc(reflect.TypeOf(b), "PutACL",
		func(ctx context.Context, opt *cos.BucketPutACLOptions) (*cos.Response, error) {
			if mockBucketPutACLFunc != nil {
				return mockBucketPutACLFunc(ctx, opt)
			}
			return nil, nil
		})
	defer patchesBucketPutACL.Reset()

	// 全局打桩 Object.PutACL（被 TestPutObjectAcl 共用）
	patchesObjectPutACL := ApplyMethodFunc(reflect.TypeOf(o), "PutACL",
		func(ctx context.Context, name string, opt *cos.ObjectPutACLOptions, id ...string) (*cos.Response, error) {
			if mockObjectPutACLFunc != nil {
				return mockObjectPutACLFunc(ctx, name, opt, id...)
			}
			return nil, nil
		})
	defer patchesObjectPutACL.Reset()

	// 全局打桩 Object.GetACL（被 TestGetObjectAcl 共用）
	patchesObjectGetACL := ApplyMethodFunc(reflect.TypeOf(o), "GetACL",
		func(ctx context.Context, name string, id ...string) (*cos.ObjectGetACLResult, *cos.Response, error) {
			if mockObjectGetACLFunc != nil {
				return mockObjectGetACLFunc(ctx, name, id...)
			}
			return nil, nil, nil
		})
	defer patchesObjectGetACL.Reset()

	// 全局打桩 Object.PutSymlink（被 TestCreateSymlink 共用）
	patchesObjectPutSymlink := ApplyMethodFunc(reflect.TypeOf(o), "PutSymlink",
		func(ctx context.Context, name string, opt *cos.ObjectPutSymlinkOptions) (*cos.Response, error) {
			if mockPutSymlinkFunc != nil {
				return mockPutSymlinkFunc(ctx, name, opt)
			}
			return nil, nil
		})
	defer patchesObjectPutSymlink.Reset()

	// 全局打桩 Object.GetSymlink（被 TestGetSymlink 共用）
	patchesObjectGetSymlink := ApplyMethodFunc(reflect.TypeOf(o), "GetSymlink",
		func(ctx context.Context, name string, opt *cos.ObjectGetSymlinkOptions) (string, *cos.Response, error) {
			if mockGetSymlinkFunc != nil {
				return mockGetSymlinkFunc(ctx, name, opt)
			}
			return "", nil, nil
		})
	defer patchesObjectGetSymlink.Reset()

	// 全局打桩 Bucket.GetVersioning（被 TestGetBucketVersioning 共用）
	patchesBucketGetVersioning := ApplyMethodFunc(reflect.TypeOf(b), "GetVersioning",
		func(ctx context.Context) (*cos.BucketGetVersionResult, *cos.Response, error) {
			if mockBucketGetVersioningFunc != nil {
				return mockBucketGetVersioningFunc(ctx)
			}
			return nil, nil, nil
		})
	defer patchesBucketGetVersioning.Reset()

	// 全局打桩 Bucket.PutVersioning（被 TestPutBucketVersioning 共用）
	patchesBucketPutVersioning := ApplyMethodFunc(reflect.TypeOf(b), "PutVersioning",
		func(ctx context.Context, opt *cos.BucketPutVersionOptions) (*cos.Response, error) {
			if mockBucketPutVersioningFunc != nil {
				return mockBucketPutVersioningFunc(ctx, opt)
			}
			return nil, nil
		})
	defer patchesBucketPutVersioning.Reset()

	// 全局打桩 Bucket.PutTagging（被 TestPutBucketTagging 共用）
	patchesBucketPutTagging := ApplyMethodFunc(reflect.TypeOf(b), "PutTagging",
		func(ctx context.Context, opt *cos.BucketPutTaggingOptions) (*cos.Response, error) {
			if mockBucketPutTaggingFunc != nil {
				return mockBucketPutTaggingFunc(ctx, opt)
			}
			return nil, nil
		})
	defer patchesBucketPutTagging.Reset()

	// 全局打桩 Bucket.GetTagging（被 TestGetBucketTagging 共用）
	patchesBucketGetTagging := ApplyMethodFunc(reflect.TypeOf(b), "GetTagging",
		func(ctx context.Context) (*cos.BucketGetTaggingResult, *cos.Response, error) {
			if mockBucketGetTaggingFunc != nil {
				return mockBucketGetTaggingFunc(ctx)
			}
			return nil, nil, nil
		})
	defer patchesBucketGetTagging.Reset()

	// 全局打桩 Bucket.DeleteTagging（被 TestDeleteBucketTagging 共用）
	patchesBucketDeleteTagging := ApplyMethodFunc(reflect.TypeOf(b), "DeleteTagging",
		func(ctx context.Context) (*cos.Response, error) {
			if mockBucketDeleteTaggingFunc != nil {
				return mockBucketDeleteTaggingFunc(ctx)
			}
			return nil, nil
		})
	defer patchesBucketDeleteTagging.Reset()

	// 全局打桩 Object.PutTagging（被 TestPutObjectTagging 共用）
	patchesObjectPutTagging := ApplyMethodFunc(reflect.TypeOf(o), "PutTagging",
		func(ctx context.Context, name string, opt *cos.ObjectPutTaggingOptions, id ...string) (*cos.Response, error) {
			if mockObjectPutTaggingFunc != nil {
				return mockObjectPutTaggingFunc(ctx, name, opt, id...)
			}
			return nil, nil
		})
	defer patchesObjectPutTagging.Reset()

	// 全局打桩 Object.GetTagging（被 TestGetObjectTagging 共用）
	patchesObjectGetTagging := ApplyMethodFunc(reflect.TypeOf(o), "GetTagging",
		func(ctx context.Context, name string, opt ...interface{}) (*cos.ObjectGetTaggingResult, *cos.Response, error) {
			if mockObjectGetTaggingFunc != nil {
				return mockObjectGetTaggingFunc(ctx, name, opt...)
			}
			return nil, nil, nil
		})
	defer patchesObjectGetTagging.Reset()

	// 全局打桩 Object.DeleteTagging（被 TestDeleteObjectTagging 共用）
	patchesObjectDeleteTagging := ApplyMethodFunc(reflect.TypeOf(o), "DeleteTagging",
		func(ctx context.Context, name string, opt ...interface{}) (*cos.Response, error) {
			if mockObjectDeleteTaggingFunc != nil {
				return mockObjectDeleteTaggingFunc(ctx, name, opt...)
			}
			return nil, nil
		})
	defer patchesObjectDeleteTagging.Reset()

	// 全局打桩 Object.Get（被 TestCatObject 共用）
	patchesObjectGet := ApplyMethodFunc(reflect.TypeOf(o), "Get",
		func(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
			if mockObjectGetFunc != nil {
				return mockObjectGetFunc(ctx, name, opt, id...)
			}
			return nil, nil
		})
	defer patchesObjectGet.Reset()

	// 全局打桩 Object.PostRestore（被 TestRestoreObjects 共用）
	patchesObjectPostRestore := ApplyMethodFunc(reflect.TypeOf(o), "PostRestore",
		func(ctx context.Context, name string, opt *cos.ObjectRestoreOptions, id ...string) (*cos.Response, error) {
			if mockObjectPostRestoreFunc != nil {
				return mockObjectPostRestoreFunc(ctx, name, opt, id...)
			}
			return nil, nil
		})
	defer patchesObjectPostRestore.Reset()

	// 全局打桩 Bucket.PutPolicy（被 TestPutBucketPolicy 共用）
	patchesBucketPutPolicy := ApplyMethodFunc(reflect.TypeOf(b), "PutPolicy",
		func(ctx context.Context, opt *cos.BucketPutPolicyOptions) (*cos.Response, error) {
			if mockBucketPutPolicyFunc != nil {
				return mockBucketPutPolicyFunc(ctx, opt)
			}
			return nil, nil
		})
	defer patchesBucketPutPolicy.Reset()

	// 全局打桩 Bucket.GetPolicy（被 TestGetBucketPolicy 共用）
	patchesBucketGetPolicy := ApplyMethodFunc(reflect.TypeOf(b), "GetPolicy",
		func(ctx context.Context) (*cos.BucketGetPolicyResult, *cos.Response, error) {
			if mockBucketGetPolicyFunc != nil {
				return mockBucketGetPolicyFunc(ctx)
			}
			return nil, nil, nil
		})
	defer patchesBucketGetPolicy.Reset()

	// 全局打桩 Bucket.DeletePolicy（被 TestDeleteBucketPolicy 共用）
	patchesBucketDeletePolicy := ApplyMethodFunc(reflect.TypeOf(b), "DeletePolicy",
		func(ctx context.Context) (*cos.Response, error) {
			if mockBucketDeletePolicyFunc != nil {
				return mockBucketDeletePolicyFunc(ctx)
			}
			return nil, nil
		})
	defer patchesBucketDeletePolicy.Reset()

	// 全局打桩 Bucket.GetEncryption（被 TestGetBucketEncryption 共用）
	patchesBucketGetEncryption := ApplyMethodFunc(reflect.TypeOf(b), "GetEncryption",
		func(ctx context.Context) (*cos.BucketGetEncryptionResult, *cos.Response, error) {
			if mockBucketGetEncryptionFunc != nil {
				return mockBucketGetEncryptionFunc(ctx)
			}
			return nil, nil, nil
		})
	defer patchesBucketGetEncryption.Reset()

	// 全局打桩 Bucket.PutEncryption（被 TestPutBucketEncryption 共用）
	patchesBucketPutEncryption := ApplyMethodFunc(reflect.TypeOf(b), "PutEncryption",
		func(ctx context.Context, opt *cos.BucketPutEncryptionOptions) (*cos.Response, error) {
			if mockBucketPutEncryptionFunc != nil {
				return mockBucketPutEncryptionFunc(ctx, opt)
			}
			return nil, nil
		})
	defer patchesBucketPutEncryption.Reset()

	// 全局打桩 Bucket.DeleteEncryption（被 TestDeleteBucketEncryption 共用）
	patchesBucketDeleteEncryption := ApplyMethodFunc(reflect.TypeOf(b), "DeleteEncryption",
		func(ctx context.Context) (*cos.Response, error) {
			if mockBucketDeleteEncryptionFunc != nil {
				return mockBucketDeleteEncryptionFunc(ctx)
			}
			return nil, nil
		})
	defer patchesBucketDeleteEncryption.Reset()

	// 全局打桩 Bucket.PutInventory（被 TestPutBucketInventory 共用）
	patchesBucketPutInventory := ApplyMethodFunc(reflect.TypeOf(b), "PutInventory",
		func(ctx context.Context, id string, opt *cos.BucketPutInventoryOptions) (*cos.Response, error) {
			if mockBucketPutInventoryFunc != nil {
				return mockBucketPutInventoryFunc(ctx, id, opt)
			}
			return nil, nil
		})
	defer patchesBucketPutInventory.Reset()

	// 全局打桩 Bucket.GetInventory（被 TestGetBucketInventory 共用）
	patchesBucketGetInventory := ApplyMethodFunc(reflect.TypeOf(b), "GetInventory",
		func(ctx context.Context, id string) (*cos.BucketGetInventoryResult, *cos.Response, error) {
			if mockBucketGetInventoryFunc != nil {
				return mockBucketGetInventoryFunc(ctx, id)
			}
			return nil, nil, nil
		})
	defer patchesBucketGetInventory.Reset()

	// 全局打桩 Bucket.DeleteInventory（被 TestDeleteBucketInventory 共用）
	patchesBucketDeleteInventory := ApplyMethodFunc(reflect.TypeOf(b), "DeleteInventory",
		func(ctx context.Context, id string) (*cos.Response, error) {
			if mockBucketDeleteInventoryFunc != nil {
				return mockBucketDeleteInventoryFunc(ctx, id)
			}
			return nil, nil
		})
	defer patchesBucketDeleteInventory.Reset()

	// 全局打桩 Bucket.PostInventory（被 TestPostBucketInventory 共用）
	patchesBucketPostInventory := ApplyMethodFunc(reflect.TypeOf(b), "PostInventory",
		func(ctx context.Context, id string, opt *cos.BucketPostInventoryOptions) (*cos.Response, error) {
			if mockBucketPostInventoryFunc != nil {
				return mockBucketPostInventoryFunc(ctx, id, opt)
			}
			return nil, nil
		})
	defer patchesBucketPostInventory.Reset()

	// 全局打桩 Bucket.ListInventoryConfigurations（被 TestListBucketInventory 共用）
	patchesBucketListInventory := ApplyMethodFunc(reflect.TypeOf(b), "ListInventoryConfigurations",
		func(ctx context.Context, token string) (*cos.ListBucketInventoryConfigResult, *cos.Response, error) {
			if mockBucketListInventoryFunc != nil {
				return mockBucketListInventoryFunc(ctx, token)
			}
			return nil, nil, nil
		})
	defer patchesBucketListInventory.Reset()

	// 全局打桩 Service.Get（被 TestGetBucketsList 共用）
	patchesServiceGet := ApplyMethodFunc(reflect.TypeOf(svc), "Get",
		func(ctx context.Context, opt ...*cos.ServiceGetOptions) (*cos.ServiceGetResult, *cos.Response, error) {
			if mockServiceGetFunc != nil {
				var o *cos.ServiceGetOptions
				if len(opt) > 0 {
					o = opt[0]
				}
				return mockServiceGetFunc(ctx, o)
			}
			return nil, nil, nil
		})
	defer patchesServiceGet.Reset()

	// 全局打桩 Bucket.ListMultipartUploads（被 TestListUploads 等共用）
	patchesBucketListMultipartUploads := ApplyMethodFunc(reflect.TypeOf(b), "ListMultipartUploads",
		func(ctx context.Context, opt *cos.ListMultipartUploadsOptions) (*cos.ListMultipartUploadsResult, *cos.Response, error) {
			if mockBucketListMultipartUploadsFunc != nil {
				return mockBucketListMultipartUploadsFunc(ctx, opt)
			}
			return nil, nil, nil
		})
	defer patchesBucketListMultipartUploads.Reset()

	// 全局打桩 Object.AbortMultipartUpload（被 TestAbortUploads 等共用）
	patchesObjectAbortMultipartUpload := ApplyMethodFunc(reflect.TypeOf(o), "AbortMultipartUpload",
		func(ctx context.Context, name, uploadID string, opt ...*cos.AbortMultipartUploadOptions) (*cos.Response, error) {
			if mockObjectAbortMultipartUploadFunc != nil {
				return mockObjectAbortMultipartUploadFunc(ctx, name, uploadID)
			}
			return nil, nil
		})
	defer patchesObjectAbortMultipartUpload.Reset()

	// 全局打桩 Object.ListParts（被 TestGetPartsListForLs、TestListParts 等共用）
	patchesObjectListParts := ApplyMethodFunc(reflect.TypeOf(o), "ListParts",
		func(ctx context.Context, name, uploadID string, opt *cos.ObjectListPartsOptions) (*cos.ObjectListPartsResult, *cos.Response, error) {
			if mockObjectListPartsFunc != nil {
				return mockObjectListPartsFunc(ctx, name, uploadID, opt)
			}
			return nil, nil, nil
		})
	defer patchesObjectListParts.Reset()

	// 全局打桩 Object.MultiCopy（被 TestCosCopy 等共用）
	patchesObjectMultiCopy := ApplyMethodFunc(reflect.TypeOf(o), "MultiCopy",
		func(ctx context.Context, key, sourceURL string, opt *cos.MultiCopyOptions, id ...string) (*cos.ObjectCopyResult, *cos.Response, error) {
			if mockObjectMultiCopyFunc != nil {
				return mockObjectMultiCopyFunc(ctx, key, sourceURL, opt, id...)
			}
			return nil, nil, nil
		})
	defer patchesObjectMultiCopy.Reset()

	// 全局打桩 Object.Download（被 TestDownload 等共用）
	patchesObjectDownload := ApplyMethodFunc(reflect.TypeOf(o), "Download",
		func(ctx context.Context, name, localPath string, opt *cos.MultiDownloadOptions, id ...string) (*cos.Response, error) {
			if mockObjectDownloadFunc != nil {
				return mockObjectDownloadFunc(ctx, name, localPath, opt, id...)
			}
			return nil, nil
		})
	defer patchesObjectDownload.Reset()

	// 全局打桩 Object.Upload（被 TestSingleUpload 等共用）
	patchesObjectUpload := ApplyMethodFunc(reflect.TypeOf(o), "Upload",
		func(ctx context.Context, key, localPath string, opt *cos.MultiUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error) {
			if mockObjectUploadFunc != nil {
				return mockObjectUploadFunc(ctx, key, localPath, opt)
			}
			return nil, nil, nil
		})
	defer patchesObjectUpload.Reset()

	// 全局打桩 Object.Put（被 TestSingleUpload 等共用）
	patchesObjectPut := ApplyMethodFunc(reflect.TypeOf(o), "Put",
		func(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
			if mockObjectPutFunc != nil {
				return mockObjectPutFunc(ctx, name, r, opt)
			}
			return nil, nil
		})
	defer patchesObjectPut.Reset()

	m.Run()
}
