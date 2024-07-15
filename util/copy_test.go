package util

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestCosCopy_single_true(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "test/cos",
	}
	fo := &FileOperations{
		Monitor: &FileProcessMonitor{},
		ErrOutput: &ErrOutput{
			outputFile: nil,
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	// getHead success
	monkey.Patch(getHead, func(*cos.Client, string) (*cos.Response, error) {
		resp := &cos.Response{
			Response: &http.Response{
				ContentLength: 1,
				Header:        http.Header{},
			},
		}
		resp.Header.Add("Last-Modified", "test")
		return resp, nil
	})
	defer monkey.UnpatchAll()
	monkey.Patch(singleCopy, func(*cos.Client, *cos.Client, *FileOperations, objectInfoType, StorageUrl, StorageUrl) (bool, error, bool, int64, string) {
		return false, nil, false, 0, ""
	})
	CosCopy(srcclient, destclient, srcurl, desturl, fo)

}

// func TestCosCopy_single_false(t *testing.T) {
// 	var srcurl StorageUrl = &CosUrl{
// 		Object: "test/cos",
// 	}
// 	fo := &FileOperations{
// 		Monitor: &FileProcessMonitor{},
// 	}
// 	secclient := &cos.Client{}
// 	destclient := &cos.Client{}
// 	var desturl StorageUrl = &CosUrl{}
// 	// getHead success
// 	monkey.Patch(getHead, func(*cos.Client, string) (*cos.Response, error) {
// 		resp := &cos.Response{
// 			Response: &http.Response{
// 				ContentLength: 1,
// 				Header:        http.Header{},
// 			},
// 		}
// 		resp.StatusCode = 404
// 		resp.Header.Add("Last-Modified", "test")
// 		return resp, errors.New("testerror")
// 	})
// 	defer monkey.UnpatchAll()
// 	CosCopy(secclient, destclient, srcurl, desturl, fo)
// 	monkey.Patch(getHead, func(*cos.Client, string) (*cos.Response, error) {
// 		resp := &cos.Response{
// 			Response: &http.Response{
// 				ContentLength: 1,
// 				Header:        http.Header{},
// 			},
// 		}
// 		resp.StatusCode = 403
// 		resp.Header.Add("Last-Modified", "test")
// 		return resp, errors.New("testerror")
// 	})
// 	CosCopy(secclient, destclient, srcurl, desturl, fo)
// }

func TestCosCopy_muti(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
	}
	fo := &FileOperations{
		Monitor: &FileProcessMonitor{},
		ErrOutput: &ErrOutput{
			outputFile: nil,
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	monkey.Patch(batchCopyFiles, func(*cos.Client, *cos.Client, StorageUrl, StorageUrl, *FileOperations) {
		fmt.Println("batchCopyFiles success")
	})
	defer monkey.UnpatchAll()
	CosCopy(srcclient, destclient, srcurl, desturl, fo)
}

func TestBatchCopyFiles_true(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
	}
	fo := &FileOperations{
		Monitor: &FileProcessMonitor{},
		Operation: Operation{
			Routines:   1,
			FailOutput: false,
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	monkey.Patch(copyFiles, func(srcClient *cos.Client, destClient *cos.Client, srcUrl, destUrl StorageUrl, fo *FileOperations, chObjects <-chan objectInfoType, chError chan<- error) {
		chError <- nil
	})
	defer monkey.UnpatchAll()
	monkey.Patch(getCosObjectList, func(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, chError chan<- error, fo *FileOperations, scanSizeNum bool, withFinishSignal bool) {
		if withFinishSignal {
			chError <- nil
		}
	})
	batchCopyFiles(srcclient, destclient, srcurl, desturl, fo)
}

func TestBatchCopyFiles_false(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
	}
	fo := &FileOperations{
		Monitor: &FileProcessMonitor{},
		Operation: Operation{
			Routines:   1,
			FailOutput: true,
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	monkey.Patch(copyFiles, func(srcClient *cos.Client, destClient *cos.Client, srcUrl, destUrl StorageUrl, fo *FileOperations, chObjects <-chan objectInfoType, chError chan<- error) {
		chError <- errors.New("testerror in copyFiles")
		chError <- nil
	})
	defer monkey.UnpatchAll()
	monkey.Patch(getCosObjectList, func(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, chError chan<- error, fo *FileOperations, scanSizeNum bool, withFinishSignal bool) {
		if withFinishSignal {
			chError <- errors.New("testerror in getCosObjectList")
		}
	})
	monkey.Patch(writeError, func(errString string, fo *FileOperations) {
		fmt.Println(errString)
	})
	batchCopyFiles(srcclient, destclient, srcurl, desturl, fo)
}

func TestCopyFiles_true(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
	}
	fo := &FileOperations{
		Monitor: &FileProcessMonitor{},
		Operation: Operation{
			Routines:    1,
			FailOutput:  true,
			ErrRetryNum: 0,
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	chError := make(chan error, fo.Operation.Routines)
	chObjects := make(chan objectInfoType, ChannelSize)
	chProgressSignal = make(chan chProgressSignalType, 10)
	chObjects <- objectInfoType{}
	monkey.Patch(singleCopy, func(*cos.Client, *cos.Client, *FileOperations, objectInfoType, StorageUrl, StorageUrl) (bool, error, bool, int64, string) {
		close(chObjects)
		return false, nil, false, 0, ""
	})
	defer monkey.UnpatchAll()
	copyFiles(srcclient, destclient, srcurl, desturl, fo, chObjects, chError)
	got := <-chError
	assert.Equal(t, got, nil, "they shoulb be equal")
}

func TestCopyFiles_false(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
	}
	fo := &FileOperations{
		Monitor: &FileProcessMonitor{},
		Operation: Operation{
			Routines:         2,
			FailOutput:       true,
			ErrRetryNum:      1,
			ErrRetryInterval: 0,
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	chError := make(chan error, fo.Operation.Routines)
	chObjects := make(chan objectInfoType, ChannelSize)
	chProgressSignal = make(chan chProgressSignalType, 10)
	chObjects <- objectInfoType{}
	var first bool = true
	monkey.Patch(singleCopy, func(*cos.Client, *cos.Client, *FileOperations, objectInfoType, StorageUrl, StorageUrl) (bool, error, bool, int64, string) {
		if first {
			close(chObjects)
			first = false
		}
		return false, errors.New("testerror in singleCopy"), false, 0, ""
	})
	defer monkey.UnpatchAll()
	copyFiles(srcclient, destclient, srcurl, desturl, fo, chObjects, chError)
	<-chError
	got := <-chError
	assert.Equal(t, got, nil, "they shoulb be equal")
}

func TestCopyFiles_false_onesecond(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
	}
	fo := &FileOperations{
		Monitor: &FileProcessMonitor{},
		Operation: Operation{
			Routines:         2,
			FailOutput:       true,
			ErrRetryNum:      1,
			ErrRetryInterval: 1,
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	chError := make(chan error, fo.Operation.Routines)
	chObjects := make(chan objectInfoType, ChannelSize)
	chProgressSignal = make(chan chProgressSignalType, 10)
	chObjects <- objectInfoType{}
	var first bool = true
	monkey.Patch(singleCopy, func(*cos.Client, *cos.Client, *FileOperations, objectInfoType, StorageUrl, StorageUrl) (bool, error, bool, int64, string) {
		if first {
			close(chObjects)
			first = false
		}
		return false, errors.New("testerror in singleCopy"), false, 0, ""
	})
	defer monkey.UnpatchAll()
	copyFiles(srcclient, destclient, srcurl, desturl, fo, chObjects, chError)
	<-chError
	got := <-chError
	assert.Equal(t, got, nil, "they shoulb be equal")
}

func TestSingleCopy_true(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
		Bucket: "src",
	}
	fo := &FileOperations{
		Config: &Config{},
		Param:  &Param{},
		Operation: Operation{
			Meta: Meta{
				CacheControl: "test",
			},
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{
		Bucket: "dest",
	}
	objectInfo := objectInfoType{
		relativeKey: "test",
		size:        10,
	}
	monkey.Patch(getCosUrl, func(bucket string, object string) string {
		return bucket + object
	})
	defer monkey.UnpatchAll()
	monkey.Patch(GenURL, func(*Config, *Param, string) *cos.BaseURL {
		url := &cos.BaseURL{
			BucketURL: &url.URL{
				Host: "test",
			},
		}
		return url
	})
	var c *cos.ObjectService
	monkey.PatchInstanceMethod(reflect.TypeOf(c), "MultiCopy", func(*cos.ObjectService, context.Context, string, string, *cos.MultiCopyOptions, ...string) (*cos.ObjectCopyResult, *cos.Response, error) {
		return nil, nil, nil
	})
	_, _, _, _, msg := singleCopy(srcclient, destclient, fo, objectInfo, srcurl, desturl)
	assert.Equal(t, msg, "\nCopy srctest to desttest")
}

func TestSingleCopy_false(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{
		Object: "",
		Bucket: "src",
	}
	fo := &FileOperations{
		Config: &Config{},
		Param:  &Param{},
		Operation: Operation{
			Meta: Meta{
				CacheControl: "test",
			},
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{
		Bucket: "dest",
	}
	objectInfo := objectInfoType{
		relativeKey: "test",
		size:        10,
	}
	monkey.Patch(getCosUrl, func(bucket string, object string) string {
		return bucket + object
	})
	defer monkey.UnpatchAll()
	monkey.Patch(GenURL, func(*Config, *Param, string) *cos.BaseURL {
		url := &cos.BaseURL{
			BucketURL: &url.URL{
				Host: "test",
			},
		}
		return url
	})
	var c *cos.ObjectService
	monkey.PatchInstanceMethod(reflect.TypeOf(c), "MultiCopy", func(*cos.ObjectService, context.Context, string, string, *cos.MultiCopyOptions, ...string) (*cos.ObjectCopyResult, *cos.Response, error) {
		return nil, nil, errors.New("test failed")
	})
	_, rErr, _, _, _ := singleCopy(srcclient, destclient, fo, objectInfo, srcurl, desturl)
	assert.Equal(t, rErr.Error(), "test failed")
}

func TestSingleCopy_folder(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{}
	fo := &FileOperations{
		Config: &Config{},
		Param:  &Param{},
		Operation: Operation{
			Meta: Meta{},
		},
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	// is a folder
	objectInfo := objectInfoType{
		relativeKey: "test/",
		size:        0,
	}
	monkey.Patch(getCosUrl, func(bucket string, object string) string {
		return bucket + object
	})
	defer monkey.UnpatchAll()
	monkey.Patch(GenURL, func(*Config, *Param, string) *cos.BaseURL {
		url := &cos.BaseURL{
			BucketURL: &url.URL{
				Host: "test",
			},
		}
		return url
	})
	var c *cos.ObjectService
	monkey.PatchInstanceMethod(reflect.TypeOf(c), "MultiCopy", func(*cos.ObjectService, context.Context, string, string, *cos.MultiCopyOptions, ...string) (*cos.ObjectCopyResult, *cos.Response, error) {
		return nil, nil, errors.New("test failed")
	})
	_, _, isDir, _, _ := singleCopy(srcclient, destclient, fo, objectInfo, srcurl, desturl)
	assert.Equal(t, isDir, true)
}

func TestSingleCopy_sync_true(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{}
	fo := &FileOperations{
		Config: &Config{},
		Param:  &Param{},
		Operation: Operation{
			Meta: Meta{},
		},
		Command: CommandSync,
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	// is a folder
	objectInfo := objectInfoType{
		relativeKey: "test",
		size:        10,
	}
	monkey.Patch(getCosUrl, func(bucket string, object string) string {
		return bucket + object
	})
	defer monkey.UnpatchAll()
	monkey.Patch(skipCopy, func(srcClient *cos.Client, destClient *cos.Client, object string, destPath string) (bool, error) {
		return true, nil
	})

	_, rErr, _, _, _ := singleCopy(srcclient, destclient, fo, objectInfo, srcurl, desturl)
	assert.Equal(t, rErr, nil)
}

func TestSingleCopy_sync_false(t *testing.T) {
	var srcurl StorageUrl = &CosUrl{}
	fo := &FileOperations{
		Config: &Config{},
		Param:  &Param{},
		Operation: Operation{
			Meta: Meta{},
		},
		Command: CommandSync,
	}
	srcclient := &cos.Client{}
	destclient := &cos.Client{}
	var desturl StorageUrl = &CosUrl{}
	// is a folder
	objectInfo := objectInfoType{
		relativeKey: "test",
		size:        10,
	}
	monkey.Patch(getCosUrl, func(bucket string, object string) string {
		return bucket + object
	})
	defer monkey.UnpatchAll()
	monkey.Patch(skipCopy, func(srcClient *cos.Client, destClient *cos.Client, object string, destPath string) (bool, error) {
		return true, errors.New("test sync false")
	})

	_, rErr, _, _, _ := singleCopy(srcclient, destclient, fo, objectInfo, srcurl, desturl)
	assert.Equal(t, rErr.Error(), "test sync false")
}
