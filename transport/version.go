package transport

type VersionInfo struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
	Build    string `json:"build"`
}

var Version VersionInfo = VersionInfo{}

func SetVersion(version, revision, build string) {
	Version = VersionInfo{
		Version:  version,
		Revision: revision,
		Build:    build,
	}
}
