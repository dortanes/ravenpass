package api

// Capabilities are the features this host offers beyond the vault.
type Capabilities struct {
	// Extensions reports that browser extensions can link to this host.
	Extensions bool `json:"extensions"`
	// Shortcuts reports that the interface takes keyboard shortcuts.
	Shortcuts bool `json:"shortcuts"`
	// DockIcon reports that the host has a Dock icon it can hide with its window.
	DockIcon bool `json:"dockIcon"`
	// StorageLocations reports that the owner can open a vault file and choose where a new one lives.
	StorageLocations bool `json:"storageLocations"`
	// SaveFiles reports that the host saves files where the owner chooses; Host.Saver sets it.
	SaveFiles bool `json:"saveFiles"`
	// UnlockOnShow reports that the locked screen asks for device unlock as the app comes into view.
	UnlockOnShow bool `json:"unlockOnShow"`
	// InterfaceSize reports that the owner sets the interface size in settings.
	InterfaceSize bool `json:"interfaceSize"`
	// Appearance reports that the host's windows follow the light, dark or system appearance chosen in settings.
	Appearance bool `json:"appearance"`
	// PhotoPicker reports a photo picker apart from the file picker PDF scans still come from.
	PhotoPicker bool `json:"photoPicker"`
	// IdentityList reports that the system can keep the vault's accounts to suggest in its own autofill.
	IdentityList bool `json:"identityList"`
	// CopyScans reports that the host puts a scan on the clipboard.
	CopyScans bool `json:"copyScans"`
	// LockWhenHidden reports that the automatic lock counts from hiding the app and can lock at once.
	LockWhenHidden bool `json:"lockWhenHidden"`
	// Screenshots reports that windows stay out of screenshots unless the owner allows them.
	Screenshots bool `json:"screenshots"`
	// SystemAutofill reports that the owner picks the autofill service and passkey provider on a system screen.
	SystemAutofill bool `json:"systemAutofill"`
	// AutoBackups reports that the host backs up the open vault into a folder the owner chooses.
	AutoBackups bool `json:"autoBackups"`
	// Print reports that the host prints through the system print dialog; Host.Printer sets it.
	Print bool `json:"print"`
	// QRCodes reports that the editor reads a one-time code setup from a QR code picture; Host.QRCodes sets it.
	QRCodes bool `json:"qrCodes"`
}

// Capabilities reports the features this host offers.
func (s *Service) Capabilities() Capabilities {
	return s.offers
}
