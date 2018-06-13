package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestStrictModeUnknownHost(t *testing.T) {
	assert.Panics(t, func() {
		classification := Classification{}
		classify(&classification, "unknown", "test_data", true)
	}, "Should panic on unknown modes in strict mode")
}

func TestStrictModeUnknownComponent(t *testing.T) {
	assert.Panics(t, func() {
		classification := Classification{}
		classify(&classification, "broken-node", "test_data", true)
	}, "Should panic on unknown component in strict mode")
}

func TestPrefixWithoutNodeSpec(t *testing.T) {
	classification := Classification{}
	err := classify(&classification, "unknown", "__missing_test_data__", false)
	assert.NotNil(t, err, "Should return an error if no nodes.yml was found")
}

func TestKnownNodeClassification(t *testing.T) {
	classification := Classification{}
	err := classify(&classification, "test-node", "test_data", false)
	assert.Nil(t, err, "Should not return an error for successful classifications")
	assert.Equal(t, "production", classification.Environment, "Should provide production environment")
	expectedData := HieraData{"bla::bla::blub": map[interface{}]interface{}{"value": "and stuff"}}
	assert.Equal(t, expectedData, classification.Data, "Data should be provided correctly")
	expectedClasses := ClassTable{"apt": ClassTableEntry{"repos": map[interface{}]interface{}{
		"backports": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
		"main": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
		"security": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
		"updates": map[interface{}]interface{}{"host": "ftp.de.debian.org"}},
	}}
	assert.Equal(t, expectedClasses, classification.Classes, "Classes should be provided correctly")
}

func TestUnknownNodeFallbackClassification(t *testing.T) {
	classification := Classification{}
	err := classify(&classification, "__unknown_node__", "test_data", false)
	assert.Nil(t, err, "No error on missing node in non-strict mode")
	assert.Equal(t, "production", classification.Environment, "Should fallback to production environment")
	expectedData := HieraData{"bla::bla::blub": map[interface{}]interface{}{"value": "and stuff"}}
	assert.Equal(t, expectedData, classification.Data, "Should fallback to correct data")
	expectedClasses := ClassTable{"apt": ClassTableEntry{"repos": map[interface{}]interface{}{
		"main": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
		"security": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
		"updates": map[interface{}]interface{}{"host": "ftp.de.debian.org"}},
	}}
	assert.Equal(t, expectedClasses, classification.Classes, "Should fallback to correct classes")
}
