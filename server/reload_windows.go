//+build windows

package chserver

// TODO(m-terel): add a different mechanism to reload API users on windows
// ReloadAPIUsers does nothing on windows since there is no SIGUSR1 on Windows.
func (al *APIListener) ReloadAPIUsers() {
}
