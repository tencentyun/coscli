package util

import (
	"testing"
)

func TestPrintCostTime(t *testing.T) {
	t.Run("正常耗时输出不 panic", func(t *testing.T) {
		// PrintCostTime 只是 fmt.Printf，验证不 panic 即可
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PrintCostTime panic: %v", r)
			}
		}()
		PrintCostTime(1000, 2500)
	})

	t.Run("开始时间等于结束时间不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PrintCostTime panic: %v", r)
			}
		}()
		PrintCostTime(1000, 1000)
	})
}

func TestPrintTransferStats(t *testing.T) {
	t.Run("ErrNum=0 且 endT>startT 时输出速度", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PrintTransferStats panic: %v", r)
			}
		}()
		fo := &FileOperations{
			Monitor: &FileProcessMonitor{
				ErrNum:       0,
				TransferSize: 1024 * 1024,
			},
			Operation: Operation{
				FailOutput: false,
			},
		}
		PrintTransferStats(1000, 2000, fo)
	})

	t.Run("ErrNum>0 且 FailOutput=true 时输出错误文件路径", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PrintTransferStats panic: %v", r)
			}
		}()
		fo := &FileOperations{
			Monitor: &FileProcessMonitor{
				ErrNum:       1,
				TransferSize: 512,
			},
			Operation: Operation{
				FailOutput:     true,
				FailOutputPath: "/tmp/coscli-test-err-output",
			},
			ErrOutput: &ErrOutput{
				Path: "/tmp/coscli-test-err-output",
			},
		}
		PrintTransferStats(1000, 2000, fo)
	})

	t.Run("endT==startT 时不输出速度（避免除零）", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PrintTransferStats panic: %v", r)
			}
		}()
		fo := &FileOperations{
			Monitor: &FileProcessMonitor{
				ErrNum:       0,
				TransferSize: 1024,
			},
			Operation: Operation{
				FailOutput: false,
			},
		}
		PrintTransferStats(1000, 1000, fo)
	})
}
