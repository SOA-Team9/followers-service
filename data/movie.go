package data

import (
	"encoding/json"
	"io"
)

type Movie struct {
	Released 	int64    `json:"released,omitempty"`
	Title    	string   `json:"title,omitempty"`
	Tagline  	string   `json:"tagline,omitempty"`
	Cast     	[]Person `json:"cast,omitempty"`
}

type Person struct {
	Name string   `json:"name"`
	Born  int64   `json:"born,omitempty"`
}

type Role struct {
	Title    	string  `json:"title"`
	Actor 		string  `json:"name"`
	Role 		string 	`json:"role"`
}

type ActorMostMovies struct {
	Actor    	string  `json:"actor"`
	Count 		int64 	`json:"count"`
}


type Movies []*Movie
type People []*Person
type Roles []*Role
type ActorsMostMovies []*ActorMostMovies

func (o *Movies) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}

func (o *People) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}

func (o *Roles) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}

func (o *ActorsMostMovies) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}

func (o *Movie) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}

func (o *Person) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}
