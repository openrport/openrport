package config

import (
	"fmt"
	"time"
)

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

func (c Config) RotationPeriod() time.Duration {
	switch c.Rotation {
	case RotationDaily:
		return 24 * time.Hour
	case RotationWeekly:
		return 7 * 24 * time.Hour
	case RotationYearly:
		return 365 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}
