package media

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func DurationFromProbe(raw []byte) string {
	var payload struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.Format.Duration == "" {
		return ""
	}
	seconds, err := strconv.ParseFloat(payload.Format.Duration, 64)
	if err != nil || seconds <= 0 {
		return ""
	}
	d := time.Duration(seconds * float64(time.Second)).Round(time.Second)
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	s := int((d % time.Minute) / time.Second)
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
