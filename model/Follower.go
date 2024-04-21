package model

import (
	"encoding/json"
	"io"
)

type Follow struct {
	FollowerID int `json:"followerID"`
	FollowedID int `json:"followedID"`
}

type Follows []*Follow

func (o *Follows) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *Follow) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}
