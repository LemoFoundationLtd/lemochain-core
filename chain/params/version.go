package params

import "fmt"

const (
	VersionMajor = 0      // Major version component of the current release
	VersionMinor = 1      // Minor version component of the current release
	VersionPatch = 0      // Patch version component of the current release
	VersionMeta  = "beta" // Version metadata to append to the version string
)

var Version = func() string {
	v := fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()
