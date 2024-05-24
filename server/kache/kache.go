package kache

import (
	"bytes"
)

type Artifact struct {
	Hash   uint64
	Length uint64
	Data   []byte
}

func (a Artifact) Equal(b Artifact) bool {
	if a.Hash != b.Hash {
		return false
	}
	return bytes.Equal(a.Data, b.Data)
}

func (a Artifact) Write(p []byte) (n int, err error) {
	return
}

func GetArtifact(url string, id string) (artifact Artifact, err error) {
	return
}

func AddArtifact(artifact Artifact, url string, id string) {
	return
}
