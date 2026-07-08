//go:build windows

package indicators

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// IF_TYPE_* values (ifdef.h) for the adapter kinds we care to distinguish.
const (
	ifTypeEthernetCSMACD = 6
	ifTypeIEEE80211      = 71
)

// IfOperStatusUp (ifdef.h).
const ifOperStatusUp = 1

// Network is the payload sent to the frontend for the network indicator.
type Network struct {
	Connected bool   `json:"connected"`
	Type      string `json:"type"` // "wifi", "ethernet", or "" when disconnected
	Name      string `json:"name"`
}

// readNetwork picks the first "up" adapter that isn't a loopback/tunnel
// pseudo-interface and reports whether it looks like Wi-Fi or Ethernet.
func readNetwork() Network {
	size := uint32(15000)
	for attempt := 0; attempt < 3; attempt++ {
		buf := make([]byte, size)
		addr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, 0, 0, addr, &size)
		if err == nil {
			return scanAdapters(addr)
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			break
		}
		// size has been updated with the required buffer length; retry.
	}
	return Network{}
}

func scanAdapters(first *windows.IpAdapterAddresses) Network {
	for a := first; a != nil; a = a.Next {
		if a.OperStatus != ifOperStatusUp {
			continue
		}
		switch a.IfType {
		case ifTypeIEEE80211:
			return Network{Connected: true, Type: "wifi", Name: windows.UTF16PtrToString(a.FriendlyName)}
		case ifTypeEthernetCSMACD:
			return Network{Connected: true, Type: "ethernet", Name: windows.UTF16PtrToString(a.FriendlyName)}
		}
	}
	return Network{}
}
