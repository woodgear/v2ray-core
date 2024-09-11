package commands

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Logf("xx")
	out_raw, err := os.ReadFile("../../x.out")
	out := string(out_raw)
	assert.NoError(t, err)
	ss, err := doParseRaw(out)
	assert.NoError(t, err)

	ss_json_raw, err := json.MarshalIndent(ss, "", " ")
	assert.NoError(t, err)
	t.Logf("out %s", ss_json_raw)
}
