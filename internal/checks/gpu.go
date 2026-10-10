package checks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

var _ model.Check = GPUSuite{}

// PCIDevice is one VGA/3D PCI device. Vendor is "nvidia", "amd" or "intel".
type PCIDevice struct {
	Vendor string
	Name   string
	Driver string
}

// GPUSuite detects GPUs via lspci with sysfs fallback, then reports per-GPU
// driver status plus CUDA when an NVIDIA driver answers.
type GPUSuite struct {
	Runner      runner.Runner
	Lspci       func(ctx context.Context) ([]byte, error)
	SysFS       func() ([]PCIDevice, error)
	GlxInfo     func(ctx context.Context) (string, error)
	NvccVersion func(ctx context.Context) (string, error)
}

func (GPUSuite) Metadata() model.CheckMetadata {
	return model.CheckMetadata{ID: "gpu-suite", Name: "GPU",
		Description: "GPU detection, drivers, and CUDA",
		Category:    model.CategoryGPU, Optional: true}
}

// parseLspciGPUs keeps lines with class [0300] or [0302], maps the PCI ID to a
// vendor, and takes the name between ": " and the last " [" (the PCI ID).
func parseLspciGPUs(out []byte) []PCIDevice {
	var devs []PCIDevice
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "[0300]") && !strings.Contains(line, "[0302]") {
			continue
		}
		var vendor string
		switch {
		case strings.Contains(line, "[10de:"):
			vendor = "nvidia"
		case strings.Contains(line, "[1002:"):
			vendor = "amd"
		case strings.Contains(line, "[8086:"):
			vendor = "intel"
		default:
			continue
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			continue
		}
		name := line[i+2:]
		if j := strings.LastIndex(name, " ["); j >= 0 {
			name = name[:j]
		}
		if name = strings.TrimSpace(name); name == "" {
			continue
		}
		devs = append(devs, PCIDevice{Vendor: vendor, Name: name})
	}
	return devs
}

// shortDeviceName shortens lspci names: trailing model text after the last
// "[...]" wins ("... [AMD/ATI] Phoenix1" -> "Phoenix1"), else the last
// non-PCI-ID bracket content ("... AD107M [GeForce RTX 4050 ...]" ->
// "GeForce RTX 4050 ..."), else the vendor-stripped name capped at 60 chars.
func shortDeviceName(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return ""
	}
	var groups []string
	lastClose := -1
	for i := 0; i < len(full); {
		o := strings.Index(full[i:], "[")
		if o < 0 {
			break
		}
		o += i
		c := strings.Index(full[o:], "]")
		if c < 0 {
			break
		}
		c += o
		groups = append(groups, full[o+1:c])
		lastClose = c
		i = c + 1
	}
	if len(groups) > 0 {
		if tail := strings.TrimSpace(full[lastClose+1:]); tail != "" && !strings.HasPrefix(tail, "(") {
			if len(tail) > 60 {
				tail = tail[:60]
			}
			return tail
		}
		for j := len(groups) - 1; j >= 0; j-- {
			if strings.Contains(groups[j], ":") {
				continue
			}
			return groups[j]
		}
	}
	s := stripVendorPrefix(full)
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func stripVendorPrefix(s string) string {
	for _, p := range []string{
		"NVIDIA Corporation ",
		"Advanced Micro Devices, Inc. [AMD/ATI] ",
		"Advanced Micro Devices, Inc. ",
		"Intel Corporation ",
		"[AMD/ATI] ",
	} {
		if strings.HasPrefix(s, p) {
			return strings.TrimSpace(s[len(p):])
		}
	}
	return s
}

func gpuVendorLabel(v string) string {
	switch v {
	case "nvidia":
		return "NVIDIA"
	case "amd":
		return "AMD"
	case "intel":
		return "Intel"
	default:
		return v
	}
}

func gpuDisplayName(d PCIDevice) string {
	if s := shortDeviceName(d.Name); s != "" {
		return s
	}
	return "GPU"
}

// parseNvidiaSMIRows parses `nvidia-smi --format=csv,noheader` output:
// "GeForce RTX 4050 ..., 550.144.03" -> {name, version} per line.
func parseNvidiaSMIRows(out string) [][2]string {
	var rows [][2]string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, ver, ok := strings.Cut(line, ",")
		if !ok {
			continue
		}
		name, ver = strings.TrimSpace(name), strings.TrimSpace(ver)
		if name == "" && ver == "" {
			continue
		}
		rows = append(rows, [2]string{name, ver})
	}
	return rows
}

// parseGlxVersion extracts "4.6" from "OpenGL version string: 4.6 (...) ...".
func parseGlxVersion(out string) (string, error) {
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "OpenGL version string:") {
			continue
		}
		_, v, _ := strings.Cut(line, ":")
		if f := strings.Fields(strings.TrimSpace(v)); len(f) > 0 {
			return f[0], nil
		}
	}
	return "", errors.New("glxinfo: no OpenGL version string")
}

// parseNvccRelease extracts "13.3" from "..., release 13.3, ...".
func parseNvccRelease(out string) (string, error) {
	i := strings.Index(out, "release ")
	if i < 0 {
		return "", errors.New("nvcc: no release found")
	}
	f := strings.Fields(out[i+len("release "):])
	if len(f) == 0 {
		return "", errors.New("nvcc: no release found")
	}
	return strings.Trim(f[0], ",;:"), nil
}

// defaultSysFSGPUs lists 0x03xx PCI devices with vendor and bound driver.
func defaultSysFSGPUs() ([]PCIDevice, error) {
	ents, err := os.ReadDir("/sys/bus/pci/devices")
	if err != nil {
		return nil, err
	}
	var devs []PCIDevice
	for _, e := range ents {
		base := filepath.Join("/sys/bus/pci/devices", e.Name())
		cls, err := os.ReadFile(filepath.Join(base, "class"))
		if err != nil || !strings.HasPrefix(strings.TrimSpace(string(cls)), "0x03") {
			continue
		}
		ven, err := os.ReadFile(filepath.Join(base, "vendor"))
		if err != nil {
			continue
		}
		var v string
		switch strings.ToLower(strings.TrimSpace(string(ven))) {
		case "0x10de", "10de":
			v = "nvidia"
		case "0x1002", "1002":
			v = "amd"
		case "0x8086", "8086":
			v = "intel"
		default:
			continue
		}
		var drv string
		if t, err := os.Readlink(filepath.Join(base, "driver")); err == nil {
			drv = filepath.Base(t)
		}
		devs = append(devs, PCIDevice{Vendor: v, Driver: drv})
	}
	return devs, nil
}

func (s GPUSuite) Run(ctx context.Context) []model.Result {
	lspci := s.Lspci
	if lspci == nil {
		lspci = func(ctx context.Context) ([]byte, error) {
			if s.Runner == nil {
				return nil, errors.New("lspci unavailable")
			}
			return s.Runner.Run(ctx, "lspci", "-nn")
		}
	}
	sysfs := s.SysFS
	if sysfs == nil {
		sysfs = defaultSysFSGPUs
	}
	glx := s.GlxInfo
	if glx == nil {
		glx = func(ctx context.Context) (string, error) {
			if s.Runner == nil {
				return "", errors.New("glxinfo unavailable")
			}
			raw, err := s.Runner.Run(ctx, "glxinfo", "-B")
			if err != nil {
				return "", err
			}
			return parseGlxVersion(string(raw))
		}
	}
	nvcc := s.NvccVersion
	if nvcc == nil {
		nvcc = func(ctx context.Context) (string, error) {
			if s.Runner == nil {
				return "", errors.New("nvcc unavailable")
			}
			raw, err := s.Runner.Run(ctx, "nvcc", "--version")
			if err != nil {
				return "", err
			}
			return parseNvccRelease(string(raw))
		}
	}

	var devs []PCIDevice
	fromSysFS := false
	if raw, err := lspci(ctx); err == nil {
		devs = parseLspciGPUs(raw)
	}
	if len(devs) == 0 {
		if sd, err := sysfs(); err == nil && len(sd) > 0 {
			devs = sd
			fromSysFS = true
		}
	}
	if len(devs) == 0 {
		return model.Single(model.Result{ID: "gpu-none", Category: model.CategoryGPU,
			Severity: model.SeverityInfo, Title: "GPU", Message: "no GPU detected"})
	}

	// lspci has no driver info: best-effort match to sysfs by vendor.
	if !fromSysFS {
		if sd, err := sysfs(); err == nil {
			used := make([]bool, len(sd))
			for i := range devs {
				for j := range sd {
					if !used[j] && sd[j].Vendor == devs[i].Vendor {
						devs[i].Driver = sd[j].Driver
						used[j] = true
						break
					}
				}
			}
		}
	}
	glVer, _ := glx(ctx)

	var rs []model.Result
	nvidiaPass := 0

	var nvDevs []PCIDevice
	for _, d := range devs {
		if d.Vendor == "nvidia" {
			nvDevs = append(nvDevs, d)
		}
	}
	if len(nvDevs) > 0 {
		var rows [][2]string
		if s.Runner != nil {
			if raw, err := s.Runner.Run(ctx, "nvidia-smi",
				"--query-gpu=name,driver_version", "--format=csv,noheader"); err == nil {
				rows = parseNvidiaSMIRows(string(raw))
			}
		}
		nvidiaResult := func(i int, name, ver string) {
			id := fmt.Sprintf("gpu-nvidia-%d", i)
			if ver == "" {
				rs = append(rs, model.Result{ID: id, Category: model.CategoryGPU,
					Severity: model.SeverityWarning, Title: "NVIDIA " + name,
					Message: "driver unavailable"})
				return
			}
			nvidiaPass++
			rs = append(rs, model.Result{ID: id, Category: model.CategoryGPU,
				Severity: model.SeverityPass, Title: "NVIDIA " + name,
				Message: "driver " + ver,
				Details: map[string]any{"name": name, "driver": ver}})
		}
		for i, d := range nvDevs {
			name := gpuDisplayName(d)
			ver := ""
			if i < len(rows) {
				if d.Name == "" {
					name = shortDeviceName(rows[i][0])
					if name == "" {
						name = "GPU"
					}
				}
				ver = rows[i][1]
			}
			nvidiaResult(i, name, ver)
		}
		for i := len(nvDevs); i < len(rows); i++ {
			name := shortDeviceName(rows[i][0])
			if name == "" {
				name = "GPU"
			}
			nvidiaResult(i, name, rows[i][1])
		}
	}

	counts := map[string]int{}
	for _, d := range devs {
		if d.Vendor != "amd" && d.Vendor != "intel" {
			continue
		}
		n := counts[d.Vendor]
		counts[d.Vendor]++
		name := gpuDisplayName(d)
		msg := "detected"
		switch {
		case d.Driver != "" && glVer != "":
			msg = d.Driver + ", GL " + glVer
		case d.Driver != "":
			msg = d.Driver
		case glVer != "":
			msg = "GL " + glVer
		}
		r := model.Result{ID: fmt.Sprintf("gpu-%s-%d", d.Vendor, n),
			Category: model.CategoryGPU, Severity: model.SeverityPass,
			Title: gpuVendorLabel(d.Vendor) + " " + name, Message: msg}
		if d.Driver != "" || glVer != "" {
			r.Details = map[string]any{"name": name, "driver": d.Driver, "gl": glVer}
		}
		rs = append(rs, r)
	}

	if nvidiaPass > 0 {
		if ver, err := nvcc(ctx); err == nil && ver != "" {
			rs = append(rs, model.Result{ID: "gpu-cuda", Category: model.CategoryGPU,
				Severity: model.SeverityInfo, Title: "CUDA", Message: ver,
				Details: map[string]any{"version": ver}})
		} else {
			rs = append(rs, model.Result{ID: "gpu-cuda", Category: model.CategoryGPU,
				Severity: model.SeverityInfo, Title: "CUDA", Message: "not installed"})
		}
	}
	return rs
}
