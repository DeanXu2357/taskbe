package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

var errInvalidCursor = errors.New("invalid cursor")

type cursor struct {
	AfterID int64 `json:"after_id"`
}

func encodeCursor(afterID int64) string {
	b, _ := json.Marshal(cursor{AfterID: afterID})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (int64, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return 0, errInvalidCursor
	}
	var c cursor
	if err := json.Unmarshal(b, &c); err != nil || c.AfterID < 1 {
		return 0, errInvalidCursor
	}
	return c.AfterID, nil
}
