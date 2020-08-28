package chshare

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func decodeLogOutput(src reflect.Type, dst reflect.Type, srcVal interface{}) (interface{}, error) {
	if src.Kind() != reflect.String {
		return srcVal, nil
	}
	if dst != reflect.TypeOf(LogOutput{}) {
		return srcVal, nil
	}
	return NewLogOutput(srcVal.(string)), nil
}

func decodeLogLevel(src reflect.Type, dst reflect.Type, srcVal interface{}) (interface{}, error) {
	if src.Kind() != reflect.String {
		return srcVal, nil
	}
	if dst != reflect.TypeOf(LogLevel(0)) {
		return srcVal, nil
	}
	return ParseLogLevel(srcVal.(string))
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

// DecodeViperConfig tries to load viper config from file and env variables
// then decoding all values into given cfg variable. cfg must be a pointer
func DecodeViperConfig(v *viper.Viper, cfg interface{}) error {
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %s", err)
		}
	}

	// ReadInConfig don't read ENV variables if keys specified in mapstructure are in lower-case
	// https://github.com/spf13/viper/issues/188
	// here is workaround to make viper read them:
	for _, key := range v.AllKeys() {
		val := v.Get(key)
		v.Set(key, val)
	}

	if err := v.Unmarshal(cfg, decoderConfigOptions...); err != nil {
		return fmt.Errorf("error parsing config file: %s", err)
	}
	return nil
}
