package model

import (
	"encoding/json"
	"io"
)

type User struct {
	Id       int    `json:"Id"`
	Username string `json:"Username"`
}

type Users []*User

func (o *User) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *User) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}
