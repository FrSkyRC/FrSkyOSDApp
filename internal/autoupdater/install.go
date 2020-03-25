package autoupdater

import "errors"

func (au *AutoUpdater) canInstallUpdates() bool {
	// For now no platform supports autoinstall
	return false
}

func (au *AutoUpdater) installUpdate(downloadedPkgPath string) error {
	return errors.New("unimplemented")
}
