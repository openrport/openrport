package auditlog

import "fmt"

const (
	RotationDaily   = "daily"
	RotationWeekly  = "weekly"
	RotationMonthly = "monthly"
	RotationYearly  = "yearly"
)

type Config struct {
	Enable           bool   `mapstructure:"enable_audit_log"`
	UseIPObfuscation bool   `mapstructure:"use_ip_obfuscation"`
	Rotation         string `mapstructure:"audit_log_rotation"`
}

func (c *Config) Validate() error {
	if !c.Enable {
		return nil
	}

	if c.Rotation != RotationDaily &&
		c.Rotation != RotationWeekly &&
		c.Rotation != RotationMonthly &&
		c.Rotation != RotationYearly {
		return fmt.Errorf("invalid api.audit_log_rotation: %q", c.Rotation)
	}

	return nil
}
