package kache

import (
	"errors"
	"fmt"
)

// Implements kache.Handler
// For testing
type Memory struct {
	Artifacts []*Artifact
	Users     []*User
}

func (m *Memory) AddUser(user *User) {
	m.Users = append(m.Users, user)
}

func (m *Memory) GetArtifact(url string, tagID string, userID uint64) (artifact *Artifact, err error) {
	user := m.Users[userID]
	if user == nil {
		return artifact, errors.New(fmt.Sprintf("Failed to get user with ID: %d", userID))
	}
	tag, ok := user.Tags[tagID]
	if !ok {
		tag = &Tag{Artifacts: make(map[string]*Artifact)}
		user.Tags[tagID] = tag
	}
	artifact, ok = tag.Artifacts[url]
	if !ok {
		return artifact, errors.New(fmt.Sprintf("Failed to get artifact for URL: %s", url))
	}
	return artifact, nil
}

func (m *Memory) AddArtifact(artifact *Artifact, url string, tagID string, userID uint64) (err error){
	user := m.Users[userID]
	if user == nil {
		return errors.New(fmt.Sprintf("Failed to get user with ID: %d", userID))
	}
	tag, ok := user.Tags[tagID]
	if !ok {
		tag = &Tag{Artifacts: make(map[string]*Artifact)}
		user.Tags[tagID] = tag
	}

	existingArtifact := false
	for _, currArtifact := range tag.Artifacts {
		if artifact.Equal(currArtifact) {
			existingArtifact = true
			break
		}
	}

	if !existingArtifact {
		m.Artifacts = append(m.Artifacts, artifact)
	}
	tag.Artifacts[url] = artifact
	return nil
}

func (m *Memory) TagArtifact(artifact *Artifact, tag string, URL string, userID uint64) () {
	m.Users[userID].Tags[tag].Artifacts[URL] = artifact
}

func CreateMemory() (m *Memory){
	a := make([]*Artifact, 0)
	u := make([]*User, 0)
	m = &Memory{Artifacts: a, Users: u}
	return m
}
