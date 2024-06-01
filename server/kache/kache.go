package kache

import (
	"bytes"
)

type Artifact struct {
	Hash   uint64
	Data   []byte
}

type Tag struct {
	// Name string
	Artifacts map[string]*Artifact
}

type User struct {
	ID uint64
	Tags map[string]*Tag
}

type Handler interface {
	GetArtifact(url string, id string, userID uint64) (artifact *Artifact, err error)
	AddArtifact(artifact *Artifact, url string, id string, userID uint64) (err error)
	AddUser(user *User)
}

func (a *Artifact) Equal(b *Artifact) bool {
	if a.Hash != b.Hash {
		return false
	}
	return bytes.Equal(a.Data, b.Data)
}

func (a Artifact) Write(p []byte) (n int, err error) {
	return
}

func CreateUser() (user *User) {
	// TODO: for now we only have an id of 0
	var id uint64
	id = 0
	tags := make(map[string]*Tag)

	user = &User{ID: id, Tags: tags}
	return user
}
