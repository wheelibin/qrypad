package keys

import (
	"log/slog"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/spf13/viper"
)

func MapCustomKeys() {
	var overrides map[string]string
	sub := viper.Sub("keys")
	if sub != nil {
		if err := sub.Unmarshal(&overrides); err != nil {
			slog.Error("error unmarshalling key overrides", "error", err)
		}
	}
	applyKeyOverrides(&DefaultKeyMap, overrides)
}

func applyKeyOverrides(km *keyMap, overrides map[string]string) {
	val := reflect.ValueOf(km).Elem()
	typ := val.Type()

	for i := range val.NumField() {
		field := typ.Field(i)
		name := strings.ToLower(field.Name)

		if overrideKey, ok := overrides[name]; ok {
			fieldVal := val.Field(i)
			if !fieldVal.CanSet() {
				continue
			}

			currentBinding, ok := fieldVal.Interface().(key.Binding)
			if !ok {
				continue
			}
			newBinding := key.NewBinding(
				key.WithKeys(overrideKey),
				key.WithHelp(overrideKey, currentBinding.Help().Desc),
			)
			fieldVal.Set(reflect.ValueOf(newBinding))
		}
	}
}
