package utils

import (
	"log/slog"
	"time"
)

// Duration is a wrapper around time.Duration supporting TOML parsing.
type Duration time.Duration

func (d Duration) Std() time.Duration { return time.Duration(d) }

func (d Duration) String() string { return time.Duration(d).String() }

func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// Secret is a string wrapper that prevents sensitive credentials (passwords, tokens, keys)
// from leaking through fmt formatting, %#v dumps, JSON serialization, or slog logging.
type Secret string

// String returns "[redacted]" to prevent leaks in %s, %v, %+v fmt calls.
func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return "[redacted]"
}

// GoString returns "[redacted]" to prevent leaks in %#v fmt calls.
func (s Secret) GoString() string {
	if s == "" {
		return `""`
	}
	return `"[redacted]"`
}

// LogValue implements slog.LogValuer so structured loggers automatically redact the secret.
func (s Secret) LogValue() slog.Value {
	if s == "" {
		return slog.StringValue("")
	}
	return slog.StringValue("[redacted]")
}

// MarshalText implements encoding.TextMarshaler to prevent accidental exposure during serialization.
func (s Secret) MarshalText() ([]byte, error) {
	if s == "" {
		return []byte(""), nil
	}
	return []byte("[redacted]"), nil
}

// UnmarshalText implements encoding.TextUnmarshaler so TOML and text decoders decode into Secret.
func (s *Secret) UnmarshalText(text []byte) error {
	*s = Secret(text)
	return nil
}

// Expose returns the underlying plaintext string when the secret is needed (e.g. authenticating).
func (s Secret) Expose() string {
	return string(s)
}
