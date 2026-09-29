package system

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

type OSInfo struct {
	OS         string `json:"os"`       // linux, windows, darwin
	Platform   string `json:"platform"` // debian, ubuntu, rocky, almalinux, centos, etc.
	Version    string `json:"version"`  // 12, 22.04, 9, etc.
	Arch       string `json:"arch"`     // x86_64, aarch64
	PrettyName string `json:"pretty_name"`
	PkgType    string `json:"pkg_type"` // deb, rpm
}

func DetectOS() OSInfo {
	info := OSInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Platform: "unknown",
		PkgType:  "unknown",
	}

	if runtime.GOARCH == "amd64" {
		info.Arch = "x86_64"
	} else if runtime.GOARCH == "arm64" {
		info.Arch = "aarch64"
	}

	if runtime.GOOS != "linux" {
		info.Platform = runtime.GOOS
		info.PrettyName = runtime.GOOS + " (" + runtime.GOARCH + ")"
		return info
	}

	// Read /etc/os-release
	file, err := os.Open("/etc/os-release")
	if err != nil {
		info.PrettyName = "Linux (" + info.Arch + ")"
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	data := make(map[string]string)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			data[key] = val
		}
	}

	id := strings.ToLower(data["ID"])
	idLike := strings.ToLower(data["ID_LIKE"])

	info.Platform = id
	info.Version = data["VERSION_ID"]
	info.PrettyName = data["PRETTY_NAME"]
	if info.PrettyName == "" {
		info.PrettyName = id + " " + info.Version
	}

	if id == "debian" || id == "ubuntu" || strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") {
		info.PkgType = "deb"
	} else if id == "centos" || id == "rhel" || id == "rocky" || id == "almalinux" || id == "fedora" || strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
		info.PkgType = "rpm"
	}

	return info
}
