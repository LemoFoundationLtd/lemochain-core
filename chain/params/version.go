package params

import "fmt"

const (
	VersionMajor = 1 // Major version component of the current release
	VersionMinor = 1 // Minor version component of the current release
	VersionPatch = 2 // Patch version component of the current release
)

var Version = func() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}()

var VersionUint = func() uint32 {
	return VersionMajor*1000000 + VersionMinor*1000 + VersionPatch
}
