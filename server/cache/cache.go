package btrfly

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
)

type Artifact struct {
	Hash string
	Data []byte
}

type Tag struct {
	// Name string
	Artifacts map[string]*Artifact
}

type User struct {
	ID   uint64
	Tags map[string]*Tag
}

type Handler interface {
	GetArtifact(url string, id string, userID uint64) (artifact *Artifact, err error)
	AddArtifact(artifact *Artifact, url string, id string, userID uint64) (err error)
	TagArtifact(artifact *Artifact, tag string, URL string, userID uint64)
	AddUser(user *User)
}

func (a *Artifact) Equal(b *Artifact) bool {
	if a.Hash != b.Hash {
		return false
	}
	return bytes.Equal(a.Data, b.Data)
}

func (a *Artifact) Write(p []byte) (n int, err error) {

	n = len(p)
	if a.Data == nil {
		a.Data = make([]byte, n)
	}

	copy(a.Data, p)

	hash := md5.Sum(a.Data)
	a.Hash = hex.EncodeToString(hash[:])

	return n, err
}

func CreateUser() (user *User) {
	// TODO: for now we only have an id of 0
	id := uint64(0)
	tags := make(map[string]*Tag)

	user = &User{ID: id, Tags: tags}
	return user
}
