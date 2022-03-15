//go:build !windows
// +build !windows

package updates

var packageManagers = []PackageManager{
	NewZypperPackageManager(),
	NewYumPackageManager(),
	NewAptPackageManager(),
}
