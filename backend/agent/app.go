package agent

// An access point for an application to retrieve information such
// as status or metrics.
type AccessPoint struct {
	Type              string
	Address           string
	Port              int64
	UseSecureProtocol bool
	Key               string
}

// Currently supported types are: "control" and "statistics".
const (
	AccessPointControl    = "control"
	AccessPointStatistics = "statistics"
)

// Base application information. This structure is embedded
// in other app specific structures like KeaApp and Bind9App.
type BaseApp struct {
	Pid          int32
	Type         string
	AccessPoints []AccessPoint
	configCaps   *configCaps
}

// Specific App like KeaApp or Bind9App have to implement
// this interface. The methods should be implemented
// in a specific way in given concrete App.
type App interface {
	GetBaseApp() *BaseApp
	DetectAllowedLogs() ([]string, error)
}

// Currently supported types are: "kea" and "bind9".
const (
	AppTypeKea   = "kea"
	AppTypeBind9 = "bind9"
)

// Creates new BaseApp instance for the given app type and with
// the given access points.
func NewBaseApp(appType string, accessPoints []AccessPoint) *BaseApp {
	return &BaseApp{
		Type:         appType,
		AccessPoints: accessPoints,
		configCaps:   newConfigCaps(),
	}
}
