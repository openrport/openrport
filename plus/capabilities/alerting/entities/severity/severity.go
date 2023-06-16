package severity

type Severity string

const (
	Information Severity = "INFORMATION"
	Warning     Severity = "WARNING"
	Average     Severity = "AVERAGE"
	High        Severity = "HIGH"
	Disaster    Severity = "DISASTER"
)
