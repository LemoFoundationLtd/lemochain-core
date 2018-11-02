package params

import "fmt"

const (
	VersionMajor = 1 // Major version component of the current release
	VersionMinor = 0 // Minor version component of the current release
	VersionPatch = 0 // Patch version component of the current release

)

var Version = func() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}()
