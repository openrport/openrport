package severity

type Severity string

const (
	Information Severity = "Information"
	Warning     Severity = "Warning"
	Average     Severity = "Average"
	High        Severity = "High"
	Disaster    Severity = "Disaster"
)
