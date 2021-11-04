package auditlog

type Status struct {
	Enabled  bool   `json:"enabled"`
	Rotation string `json:"rotation"`
}

func (a *AuditLog) Status() Status {
	return Status{
		Enabled:  a.config.Enable,
		Rotation: a.config.Rotation,
	}
}
