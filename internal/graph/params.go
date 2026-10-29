package graph

import "encoding/json"

type Params map[string]any

func (p Params) Float(key string, def float64) float64 {
	if v, ok := p[key].(float64); ok {
		return v
	}
	return def
}

func (p Params) Int(key string, def int) int {
	return int(p.Float(key, float64(def)))
}

func (p Params) String(key, def string) string {
	if v, ok := p[key].(string); ok {
		return v
	}
	return def
}

func (p Params) Bool(key string, def bool) bool {
	if v, ok := p[key].(bool); ok {
		return v
	}
	return def
}

// Decode unmarshals a structured param into out. Missing keys leave out untouched.
func (p Params) Decode(key string, out any) error {
	v, ok := p[key]
	if !ok {
		return nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
