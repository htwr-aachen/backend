package configurator

import (
	"github.com/knadh/koanf/v2"
)

// NewKoanf returns a new koanf instance.
func NewKoanf() *koanf.Koanf {
	return koanf.New(".")
}
