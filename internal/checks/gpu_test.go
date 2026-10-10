package checks

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

const gpuLspciHybrid = `01:00.0 VGA compatible controller [0300]: NVIDIA Corporation AD107M [GeForce RTX 4050 Max-Q / Mobile] [10de:28a1] (rev a1)
c5:00.0 VGA compatible controller [0300]: Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 [1002:15bf] (rev d2)
`

const gpuLspciNvidiaOnly = `01:00.0 VGA compatible controller [0300]: NVIDIA Corporation AD107M [GeForce RTX 4050 Max-Q / Mobile] [10de:28a1] (rev a1)
`

const gpuSmiSingle = "GeForce RTX 4050 Max-Q / Mobile, 550.144.03\n"

const gpuSmiMulti = `GeForce RTX 4050 Max-Q / Mobile, 550.144.03
GeForce RTX 4060 Ti, 550.144.03
`

func gpuByID(rs []model.Result, id string) *model.Result {
	for i := range rs {
		if rs[i].ID == id {
			return &rs[i]
		}
	}
	return nil
}

func gpuSmiCalls(m *runner.MockRunner) int {
	n := 0
	for _, c := range m.Calls {
		if c.Name == "nvidia-smi" {
			n++
		}
	}
	return n
}

func TestGPU_Metadata(t *testing.T) {
	md := GPUSuite{}.Metadata()
	if md.ID != "gpu-suite" || md.Name != "GPU" || md.Category != model.CategoryGPU || !md.Optional {
		t.Fatalf("bad metadata: %+v", md)
	}
}

func TestGPU_FullHybrid(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"nvidia-smi": func([]string) ([]byte, error) { return []byte(gpuSmiSingle), nil },
	}}
	s := GPUSuite{
		Runner: m,
		Lspci:  func(context.Context) ([]byte, error) { return []byte(gpuLspciHybrid), nil },
		SysFS: func() ([]PCIDevice, error) {
			return []PCIDevice{{Vendor: "amd", Driver: "amdgpu"}}, nil
		},
		GlxInfo:     func(context.Context) (string, error) { return "4.6", nil },
		NvccVersion: func(context.Context) (string, error) { return "13.3", nil },
	}
	rs := s.Run(context.Background())
	if len(rs) != 3 {
		t.Fatalf("want 3 results, got %d: %+v", len(rs), rs)
	}
	nv := gpuByID(rs, "gpu-nvidia-0")
	if nv == nil || nv.Severity != model.SeverityPass {
		t.Fatalf("nvidia should pass: %+v", rs)
	}
	if nv.Title != "NVIDIA GeForce RTX 4050 Max-Q / Mobile" {
		t.Fatalf("bad nvidia title: %q", nv.Title)
	}
	if nv.Message != "driver 550.144.03" {
		t.Fatalf("bad nvidia msg: %q", nv.Message)
	}
	amd := gpuByID(rs, "gpu-amd-0")
	if amd == nil || amd.Severity != model.SeverityPass {
		t.Fatalf("amd should pass: %+v", rs)
	}
	if amd.Title != "AMD Phoenix1" {
		t.Fatalf("bad amd title: %q", amd.Title)
	}
	if amd.Message != "amdgpu, GL 4.6" {
		t.Fatalf("bad amd msg: %q", amd.Message)
	}
	cuda := gpuByID(rs, "gpu-cuda")
	if cuda == nil || cuda.Severity != model.SeverityInfo || cuda.Message != "13.3" {
		t.Fatalf("bad cuda: %+v", rs)
	}
	if n := gpuSmiCalls(m); n != 1 {
		t.Fatalf("want 1 nvidia-smi call, got %d", n)
	}
}

func TestGPU_NvidiaNoSMI(t *testing.T) {
	cudaCalled := false
	m := &runner.MockRunner{}
	s := GPUSuite{
		Runner:      m,
		Lspci:       func(context.Context) ([]byte, error) { return []byte(gpuLspciNvidiaOnly), nil },
		SysFS:       func() ([]PCIDevice, error) { return nil, errors.New("no sysfs") },
		GlxInfo:     func(context.Context) (string, error) { return "", errors.New("no glx") },
		NvccVersion: func(context.Context) (string, error) { cudaCalled = true; return "13.3", nil },
	}
	rs := s.Run(context.Background())
	if len(rs) != 1 {
		t.Fatalf("want 1 result, got %+v", rs)
	}
	w := gpuByID(rs, "gpu-nvidia-0")
	if w == nil || w.Severity != model.SeverityWarning {
		t.Fatalf("want nvidia warning: %+v", rs)
	}
	if w.Message != "driver unavailable" {
		t.Fatalf("bad msg: %q", w.Message)
	}
	if w.Remediation != nil {
		t.Fatalf("remediation must be nil: %+v", w)
	}
	if cudaCalled || gpuByID(rs, "gpu-cuda") != nil {
		t.Fatalf("no CUDA without nvidia PASS: %+v", rs)
	}
}

func TestGPU_AMDOnly(t *testing.T) {
	cudaCalled := false
	s := GPUSuite{
		Runner: &runner.MockRunner{},
		Lspci: func(context.Context) ([]byte, error) {
			return []byte("c5:00.0 VGA compatible controller [0300]: Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 [1002:15bf] (rev d2)\n"), nil
		},
		SysFS: func() ([]PCIDevice, error) {
			return []PCIDevice{{Vendor: "amd", Driver: "amdgpu"}}, nil
		},
		GlxInfo:     func(context.Context) (string, error) { return "", errors.New("no glx") },
		NvccVersion: func(context.Context) (string, error) { cudaCalled = true; return "", errors.New("no nvcc") },
	}
	rs := s.Run(context.Background())
	amd := gpuByID(rs, "gpu-amd-0")
	if amd == nil || amd.Severity != model.SeverityPass || amd.Title != "AMD Phoenix1" {
		t.Fatalf("bad amd: %+v", rs)
	}
	if amd.Message != "amdgpu" {
		t.Fatalf("bad amd msg: %q", amd.Message)
	}
	if cudaCalled || gpuByID(rs, "gpu-cuda") != nil {
		t.Fatalf("no CUDA without nvidia: %+v", rs)
	}
}

func TestGPU_IntelOnly(t *testing.T) {
	s := GPUSuite{
		Runner: &runner.MockRunner{},
		Lspci: func(context.Context) ([]byte, error) {
			return []byte("00:02.0 VGA compatible controller [0300]: Intel Corporation Raptor Lake-P [Iris Xe Graphics] [8086:a7a0] (rev 04)\n"), nil
		},
		SysFS: func() ([]PCIDevice, error) {
			return []PCIDevice{{Vendor: "intel", Driver: "i915"}}, nil
		},
		GlxInfo:     func(context.Context) (string, error) { return "4.6", nil },
		NvccVersion: func(context.Context) (string, error) { return "", errors.New("no nvcc") },
	}
	rs := s.Run(context.Background())
	in := gpuByID(rs, "gpu-intel-0")
	if in == nil || in.Severity != model.SeverityPass {
		t.Fatalf("bad intel: %+v", rs)
	}
	if in.Title != "Intel Iris Xe Graphics" {
		t.Fatalf("bad intel title: %q", in.Title)
	}
	if in.Message != "i915, GL 4.6" {
		t.Fatalf("bad intel msg: %q", in.Message)
	}
}

func TestGPU_NoGPU(t *testing.T) {
	s := GPUSuite{
		Runner:      &runner.MockRunner{},
		Lspci:       func(context.Context) ([]byte, error) { return []byte(""), nil },
		SysFS:       func() ([]PCIDevice, error) { return nil, nil },
		GlxInfo:     func(context.Context) (string, error) { return "", errors.New("no glx") },
		NvccVersion: func(context.Context) (string, error) { return "", errors.New("no nvcc") },
	}
	rs := s.Run(context.Background())
	if len(rs) != 1 {
		t.Fatalf("want 1 result, got %+v", rs)
	}
	if rs[0].ID != "gpu-none" || rs[0].Severity != model.SeverityInfo || rs[0].Message != "no GPU detected" {
		t.Fatalf("bad no-gpu: %+v", rs[0])
	}
}

func TestGPU_LspciFailSysFSFallback(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"nvidia-smi": func([]string) ([]byte, error) { return []byte(gpuSmiSingle), nil },
	}}
	s := GPUSuite{
		Runner: m,
		Lspci:  func(context.Context) ([]byte, error) { return nil, errors.New("no lspci") },
		SysFS: func() ([]PCIDevice, error) {
			return []PCIDevice{{Vendor: "nvidia", Name: "GeForce RTX 4050 Max-Q / Mobile", Driver: "nvidia"}}, nil
		},
		GlxInfo:     func(context.Context) (string, error) { return "", errors.New("no glx") },
		NvccVersion: func(context.Context) (string, error) { return "13.3", nil },
	}
	rs := s.Run(context.Background())
	nv := gpuByID(rs, "gpu-nvidia-0")
	if nv == nil || nv.Severity != model.SeverityPass || nv.Message != "driver 550.144.03" {
		t.Fatalf("fallback nvidia should pass: %+v", rs)
	}
	if gpuByID(rs, "gpu-cuda") == nil {
		t.Fatalf("cuda expected: %+v", rs)
	}
}

func TestGPU_MultiSMI(t *testing.T) {
	lspci := `01:00.0 VGA compatible controller [0300]: NVIDIA Corporation AD107M [GeForce RTX 4050 Max-Q / Mobile] [10de:28a1] (rev a1)
02:00.0 VGA compatible controller [0300]: NVIDIA Corporation AD106 [GeForce RTX 4060 Ti] [10de:2803] (rev a1)
`
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"nvidia-smi": func([]string) ([]byte, error) { return []byte(gpuSmiMulti), nil },
	}}
	s := GPUSuite{
		Runner:      m,
		Lspci:       func(context.Context) ([]byte, error) { return []byte(lspci), nil },
		SysFS:       func() ([]PCIDevice, error) { return nil, errors.New("no sysfs") },
		GlxInfo:     func(context.Context) (string, error) { return "", errors.New("no glx") },
		NvccVersion: func(context.Context) (string, error) { return "13.3", nil },
	}
	rs := s.Run(context.Background())
	for _, id := range []string{"gpu-nvidia-0", "gpu-nvidia-1"} {
		r := gpuByID(rs, id)
		if r == nil || r.Severity != model.SeverityPass {
			t.Fatalf("%s should pass: %+v", id, rs)
		}
		if !strings.HasPrefix(r.Message, "driver 550.144.03") {
			t.Fatalf("%s bad msg: %q", id, r.Message)
		}
	}
	if n := gpuSmiCalls(m); n != 1 {
		t.Fatalf("want 1 nvidia-smi call, got %d", n)
	}
}

func TestGPU_CudaMissing(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"nvidia-smi": func([]string) ([]byte, error) { return []byte(gpuSmiSingle), nil },
	}}
	s := GPUSuite{
		Runner:      m,
		Lspci:       func(context.Context) ([]byte, error) { return []byte(gpuLspciNvidiaOnly), nil },
		SysFS:       func() ([]PCIDevice, error) { return nil, errors.New("no sysfs") },
		GlxInfo:     func(context.Context) (string, error) { return "", errors.New("no glx") },
		NvccVersion: func(context.Context) (string, error) { return "", errors.New("no nvcc") },
	}
	rs := s.Run(context.Background())
	cuda := gpuByID(rs, "gpu-cuda")
	if cuda == nil || cuda.Severity != model.SeverityInfo || cuda.Message != "not installed" {
		t.Fatalf("bad cuda: %+v", rs)
	}
}

func TestParseLspciGPUs(t *testing.T) {
	out := []byte(`01:00.0 VGA compatible controller [0300]: NVIDIA Corporation AD107M [GeForce RTX 4050 Max-Q / Mobile] [10de:28a1] (rev a1)
c5:00.0 VGA compatible controller [0300]: Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 [1002:15bf] (rev d2)
00:02.0 3D controller [0302]: Intel Corporation Raptor Lake-P [Iris Xe Graphics] [8086:a7a0] (rev 04)
02:00.0 Ethernet controller [0200]: Intel Corporation Ethernet [8086:15f3]
03:00.0 VGA compatible controller [0300]: Unknown Vendor Foo [1234:5678]
`)
	devs := parseLspciGPUs(out)
	if len(devs) != 3 {
		t.Fatalf("want 3 gpus, got %+v", devs)
	}
	if devs[0].Vendor != "nvidia" || devs[1].Vendor != "amd" || devs[2].Vendor != "intel" {
		t.Fatalf("bad vendors: %+v", devs)
	}
	if !strings.Contains(devs[0].Name, "AD107M") || !strings.Contains(devs[1].Name, "Phoenix1") {
		t.Fatalf("bad names: %+v", devs)
	}
	if len(parseLspciGPUs([]byte(""))) != 0 {
		t.Fatal("empty input must yield no devices")
	}
}

func TestShortDeviceName(t *testing.T) {
	got := shortDeviceName("NVIDIA Corporation AD107M [GeForce RTX 4050 Max-Q / Mobile]")
	if got != "GeForce RTX 4050 Max-Q / Mobile" {
		t.Fatalf("got %q", got)
	}
	if got := shortDeviceName("Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1"); got != "Phoenix1" {
		t.Fatalf("got %q", got)
	}
	long := "Intel Corporation " + strings.Repeat("x", 80)
	if got := shortDeviceName(long); len(got) != 60 {
		t.Fatalf("want 60 chars, got %q", got)
	}
}

func TestParseGlxNvcc(t *testing.T) {
	v, err := parseGlxVersion("name of display: :0\nOpenGL version string: 4.6 (Compatibility Profile) Mesa 24.2.8\n")
	if err != nil || v != "4.6" {
		t.Fatalf("got %q, %v", v, err)
	}
	if _, err := parseGlxVersion("no version here\n"); err == nil {
		t.Fatal("want error for missing GL string")
	}
	r, err := parseNvccRelease("Cuda compilation tools, release 13.3, V13.3.30\n")
	if err != nil || r != "13.3" {
		t.Fatalf("got %q, %v", r, err)
	}
	if _, err := parseNvccRelease("nothing\n"); err == nil {
		t.Fatal("want error for missing release")
	}
}
