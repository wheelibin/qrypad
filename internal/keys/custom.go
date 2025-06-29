package keys

import (
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/spf13/viper"
)

func MapCustomKeys() {
	var overrides map[string]string
	sub := viper.Sub("keys")
	if sub != nil {
		sub.Unmarshal(&overrides)
	}
	applyKeyOverrides(&DefaultKeyMap, overrides)
}

func applyKeyOverrides(km *keyMap, overrides map[string]string) {
	val := reflect.ValueOf(km).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		name := strings.ToLower(field.Name)

		if overrideKey, ok := overrides[name]; ok {
			fieldVal := val.Field(i)
			if !fieldVal.CanSet() {
				continue
			}

			currentBinding := fieldVal.Interface().(key.Binding)
			newBinding := key.NewBinding(
				key.WithKeys(overrideKey),
				key.WithHelp(overrideKey, currentBinding.Help().Desc),
			)
			fieldVal.Set(reflect.ValueOf(newBinding))
		}
	}
}
