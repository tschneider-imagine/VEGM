package pack

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadPack(path string) (*MessagePack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pack %q: %w", path, err)
	}
	return ParsePack(data)
}

func ParsePack(data []byte) (*MessagePack, error) {
	var p MessagePack
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decode pack json: %w", err)
	}
	if err := ValidatePack(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func LoadOverlay(path string) (*MessageOverlay, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read overlay %q: %w", path, err)
	}
	return ParseOverlay(data)
}

func ParseOverlay(data []byte) (*MessageOverlay, error) {
	var o MessageOverlay
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, fmt.Errorf("decode overlay json: %w", err)
	}
	if err := ValidateOverlay(&o); err != nil {
		return nil, err
	}
	return &o, nil
}
