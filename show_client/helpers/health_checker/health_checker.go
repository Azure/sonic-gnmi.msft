package helpers

const (
	StatusOK    = "OK"
	StatusNotOK = "Not OK"

	INFO_FIELD_OBJECT_TYPE   = "type"
	INFO_FIELD_OBJECT_STATUS = "status"
	INFO_FIELD_OBJECT_MSG    = "message"
)

var Summary = StatusOK

// Checker is the interface that all health checkers must satisfy.
type Checker interface {
	// Perform the check.
	// :param config: Health checker configuration.
	Check(config *Config)

	// Get category of the checker.
	// :return: String category
	GetCategory() string

	// Get information of the checker.
	// :return: Check result.
	GetInfo() map[string]interface{}

	// Str returns a human-readable name for the checker (used in error messages).
	Str() string
}

// HealthChecker is the base type for health checker. A checker is an object that
// performs system health check for a particular category, it collects and stores
// information after the check.
type HealthChecker struct {
	info map[string]interface{}
}

func NewHealthChecker() HealthChecker {
	/* NewHealthChecker creates a HealthChecker with an initialized info map. */
	return HealthChecker{
		info: make(map[string]interface{}),
	}
}

func (hc *HealthChecker) AddInfo(objectName string, key string, value string) {
	/* AddInfo adds a check result for an object.
	:param objectName: Object name.
	:param key: Object attribute name.
	:param value: Object attribute value.*/
	objectMap, ok := hc.info[objectName].(map[string]interface{})
	if !ok {
		objectMap = make(map[string]interface{})
		hc.info[objectName] = objectMap
	}
	objectMap[key] = value
}

func (hc *HealthChecker) SetObjectNotOK(objectType string, objectName string, message string) {
	/* SetObjectNotOK sets that an object is not OK.
	:param objectType: Object type.
	:param objectName: Object name.
	:param message: A message to describe what is wrong with the object.*/
	hc.AddInfo(objectName, INFO_FIELD_OBJECT_TYPE, objectType)
	hc.AddInfo(objectName, INFO_FIELD_OBJECT_MSG, message)
	hc.AddInfo(objectName, INFO_FIELD_OBJECT_STATUS, StatusNotOK)
	Summary = StatusNotOK
}

func (hc *HealthChecker) SetObjectOK(objectType string, objectName string) {
	/* SetObjectOK sets that an object is in good state.
	:param objectType: Object type.
	:param objectName: Object name.*/
	hc.AddInfo(objectName, INFO_FIELD_OBJECT_TYPE, objectType)
	hc.AddInfo(objectName, INFO_FIELD_OBJECT_MSG, "")
	hc.AddInfo(objectName, INFO_FIELD_OBJECT_STATUS, StatusOK)
}

func (hc *HealthChecker) Reset() {
	/* Reset resets the status of the checker. Called every time before the check. */
	hc.info = make(map[string]interface{})
}

func (hc *HealthChecker) GetInfo() map[string]interface{} {
	/* GetInfo returns information of the checker. A checker usually checks a few objects
	and each object status will be put to info.
	:return: Check result.*/
	return hc.info
}
