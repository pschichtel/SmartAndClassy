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
	expectedClassification := Classification{
		Classes: ClassTable{"apt": ClassTableEntry{"repos": map[interface{}]interface{}{
			"backports": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
			"main": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
			"security": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
			"updates": map[interface{}]interface{}{"host": "ftp.de.debian.org"}},
		}},
		Data: DataTable{"bla::bla::blub": map[interface{}]interface{}{"value": "and stuff"}},
		Parameters: DataTable{"such": "parameter"},
		Environment: "production",
	}
	classification := Classification{}
	err := classify(&classification, "test-node", "test_data", false)
	assert.Nil(t, err, "Should not return an error for successful classifications")
	assert.Equal(t, expectedClassification, classification, "Should classify correctly")
}

func TestUnknownNodeFallbackClassification(t *testing.T) {
	expectedClassification := Classification{
		Classes: ClassTable{"apt": ClassTableEntry{"repos": map[interface{}]interface{}{
			"main": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
			"security": map[interface{}]interface{}{"host": "ftp.de.debian.org"},
			"updates": map[interface{}]interface{}{"host": "ftp.de.debian.org"}},
		}},
		Data: DataTable{"bla::bla::blub": map[interface{}]interface{}{"value": "and stuff"}},
		Parameters: DataTable{},
		Environment: "production",
	}
	classification := Classification{}
	err := classify(&classification, "__unknown_node__", "test_data", false)
	assert.Nil(t, err, "No error on missing node in non-strict mode")
	assert.Equal(t, expectedClassification, classification, "Should fallback correctly")
}

func TestResolutionOrder(t *testing.T) {
	expectedClassification := Classification{
		Classes: ClassTable{"a": ClassTableEntry{"b": 1}},
		Data: DataTable{},
		Parameters: DataTable{},
		Environment: "production",
	}
	classification := Classification{}
	err := classify(&classification, "prio-node", "test_data", true)
	assert.Nil(t, err, "Should not return an error.")
	assert.Equal(t, expectedClassification, classification, "Should traverse the implications in post-order to prioritize values higher up the tree.")
}
