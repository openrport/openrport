//+build !windows

package updates

var packageManagers = []PackageManager{
	NewZypperPackageManager(),
	NewYumPackageManager(),
	NewAptPackageManager(),
}
