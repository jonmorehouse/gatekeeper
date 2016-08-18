package utils

import (
	"testing"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper/test"
)

type validConfig struct {
	Str     string        `flag:"str" default:"default"`
	Dur     time.Duration `flag:"duration" default:"1m"`
	Bool    bool          `flag:"bool" default:"false"`
	Uint    uint          `flag:"uint" default:"1000"`
	Int     int           `flag:"int" default:"1000"`
	Float64 float64       `flag:"float64" default:"1000.0"`
	StrList []string      `flag:"str_list" default:"c,d,e"`
}

type invalidConfig struct {
	Invalid struct{} `flag:"invalid"`
}

func TestConfig__OK(t *testing.T) {
	var cfg validConfig
	opts := map[string]interface{}{
		"str":      "str",
		"duration": "1s",
		"bool":     "true",
		"uint":     "0",
		"int":      "10",
		"float64":  "10.4",
		"str_list": "a,b,c",
	}

	test.AssertNil(t, ParseConfig(opts, &cfg))
	test.AssertEqual(t, cfg.Str, "str")
	test.AssertEqual(t, cfg.Dur, time.Second)
	test.AssertEqual(t, cfg.Bool, true)
	test.AssertEqual(t, cfg.Uint, uint(0))
	test.AssertEqual(t, cfg.Int, int(10))
	test.AssertEqual(t, cfg.Float64, float64(10.4))
	test.AssertEqual(t, cfg.StrList, []string{"a", "b", "c"})
}

func TestConfig__DefaultOK(t *testing.T) {
	var cfg validConfig
	opts := map[string]interface{}{}

	test.AssertNil(t, ParseConfig(opts, &cfg))
	test.AssertEqual(t, cfg.Str, "default")
	test.AssertEqual(t, cfg.Dur, time.Minute)
	test.AssertEqual(t, cfg.Bool, false)
	test.AssertEqual(t, cfg.Uint, uint(1000))
	test.AssertEqual(t, cfg.Int, int(1000))
	test.AssertEqual(t, cfg.Float64, float64(1000.0))
	test.AssertEqual(t, cfg.StrList, []string{"c", "d", "e"})
}

func TestConfig__InvalidType(t *testing.T) {
	opts := map[string]interface{}{
		"invalid": "invalid",
	}
	var invalid invalidConfig

	err := ParseConfig(opts, &invalid)
	test.AssertNotNil(t, err)
	test.AssertEqual(t, InvalidTypeErr, err)
}

func TestConfig__InvalidRawValue(t *testing.T) {
	opts := map[string]interface{}{
		"str": struct{}{},
	}
	var cfg validConfig

	err := ParseConfig(opts, &cfg)
	test.AssertNotNil(t, err)
	test.AssertEqual(t, InvalidRawValueErr, err)
}

func TestConfig__BoolParsing(t *testing.T) {
	opts := map[string]interface{}{
		"bool": true,
	}
	var cfg validConfig

	err := ParseConfig(opts, &cfg)
	test.AssertNil(t, err)
	test.AssertEqual(t, cfg.Bool, true)
}

func TestConfig__BoolInvalidDest(t *testing.T) {
	opts := map[string]interface{}{
		"str": true,
	}
	var cfg validConfig

	err := ParseConfig(opts, &cfg)
	test.AssertNotNil(t, err)
}
