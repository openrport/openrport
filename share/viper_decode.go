package chshare

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

func decodeLogOutput(src reflect.Type, dst reflect.Type, srcVal interface{}) (interface{}, error) {
	if src.Kind() != reflect.String {
		return srcVal, nil
	}
	if dst != reflect.TypeOf(logger.LogOutput{}) {
		return srcVal, nil
	}
	return logger.NewLogOutput(srcVal.(string)), nil
}

func decodeLogLevel(src reflect.Type, dst reflect.Type, srcVal interface{}) (interface{}, error) {
	if src.Kind() != reflect.String {
		return srcVal, nil
	}
	if dst != reflect.TypeOf(logger.LogLevel(0)) {
		return srcVal, nil
	}
	return logger.ParseLogLevel(srcVal.(string))
}

func decodeStringArray(src reflect.Type, dst reflect.Type, srcVal interface{}) (interface{}, error) {
	// workaround for a problem when viper can't parse value provided via StringArray flag
	// https://github.com/spf13/viper/issues/380
	// https://github.com/spf13/viper/pull/398

	if src.Kind() != reflect.String {
		return srcVal, nil
	}
	if dst != reflect.TypeOf([]string{}) {
		return srcVal, nil
	}

	str := srcVal.(string)
	str = strings.TrimPrefix(str, "[")
	str = strings.TrimSuffix(str, "]")
	if str == "" {
		return []string{}, nil
	}
	stringReader := strings.NewReader(str)
	csvReader := csv.NewReader(stringReader)
	return csvReader.Read()
}

var decodeHooks = mapstructure.ComposeDecodeHookFunc(
	decodeLogOutput,
	decodeLogLevel,
	decodeStringArray,
	mapstructure.StringToTimeDurationHookFunc(),
	mapstructure.StringToSliceHookFunc(","),
)
var decoderConfigOptions = []viper.DecoderConfigOption{viper.DecodeHook(decodeHooks)}

// DecodeViperConfig tries to load viper config from either a file or reader and env variables
// then decodes all values into given cfg variable. cfg must be a pointer.
func DecodeViperConfig(v *viper.Viper, cfg interface{}, cfgReader io.Reader) error {
	if cfgReader == nil {
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return fmt.Errorf("error reading config file: %s", err)
			}
		}
	} else {
		if err := v.ReadConfig(cfgReader); err != nil {
			return fmt.Errorf("error reading config contents: %w", err)
		}
	}

	readEnvVars(v)

	if err := v.Unmarshal(cfg, decoderConfigOptions...); err != nil {
		return fmt.Errorf("error parsing config file: %s", err)
	}
	return nil
}

func readEnvVars(v *viper.Viper) {
	// viper doesn't read ENV variables if keys specified in mapstructure are in lower-case
	// https://github.com/spf13/viper/issues/188
	// here is workaround to make viper read them:
	for _, key := range v.AllKeys() {
		val := v.Get(key)
		v.Set(key, val)
	}
}
